/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubernetes

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/helm/pkg/strvals"

	"github.com/hashicorp/go-version"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

const operatorName = "dapr-operator"

var crds = []string{
	"components",
	"configuration",
	"subscription",
	"resiliency",
	"httpendpoints",
}

var crdsFullResources = []string{
	"components.dapr.io",
	"configurations.dapr.io",
	"subscriptions.dapr.io",
	"resiliencies.dapr.io",
	"httpendpoints.dapr.io",
}

type UpgradeConfig struct {
	RuntimeVersion   string
	DashboardVersion string
	Args             []string
	Timeout          uint
	ImageRegistryURI string
	ImageVariant     string
}

// UpgradeOptions represents options for the upgrade function.
type UpgradeOptions struct {
	WithRetry     bool
	MaxRetries    int
	RetryInterval time.Duration
}

// UpgradeOption is a functional option type for configuring upgrade.
type UpgradeOption func(*UpgradeOptions)

func Upgrade(conf UpgradeConfig) error {
	helmRepo := utils.GetEnv("DAPR_HELM_REPO_URL", daprHelmRepo)
	status, err := GetDaprResourcesStatus()
	if err != nil {
		return err
	}

	daprVersion := GetDaprVersion(status)
	print.InfoStatusEvent(os.Stdout, "Dapr control plane version %s detected in namespace %s", daprVersion, status[0].Namespace)

	hasDashboardInDaprChart, err := IsDashboardIncluded(daprVersion)
	if err != nil {
		return err
	}

	upgradeClient, helmConf, err := newUpgradeClient(status[0].Namespace, conf)
	if err != nil {
		return fmt.Errorf("unable to create helm client: %w", err)
	}

	controlPlaneChart, err := getHelmChart(conf.RuntimeVersion, "dapr", helmRepo, helmConf)
	if err != nil {
		return fmt.Errorf("unable to get helm chart: %w", err)
	}

	willHaveDashboardInDaprChart, err := IsDashboardIncluded(conf.RuntimeVersion)
	if err != nil {
		return err
	}

	// Before we do anything, checks if installing dashboard is allowed.
	if willHaveDashboardInDaprChart && conf.DashboardVersion != "" {
		// We cannot install Dashboard separately if Dapr's chart has it already.
		return fmt.Errorf("cannot install Dashboard because Dapr version %s already contains it - installation aborted", conf.RuntimeVersion)
	}

	dashboardExists, err := confirmExist(helmConf, dashboardReleaseName)
	if err != nil {
		return err
	}

	if !hasDashboardInDaprChart && willHaveDashboardInDaprChart && dashboardExists {
		print.InfoStatusEvent(os.Stdout, "Dashboard being uninstalled prior to Dapr control plane upgrade...")
		uninstallClient := helm.NewUninstall(helmConf)
		uninstallClient.Timeout = time.Duration(conf.Timeout) * time.Second //nolint:gosec

		_, err = uninstallClient.Run(dashboardReleaseName)
		if err != nil {
			return err
		}
	}

	var dashboardChart *chart.Chart
	if conf.DashboardVersion != "" {
		dashboardChart, err = getHelmChart(conf.DashboardVersion, dashboardReleaseName, helmRepo, helmConf)
		if err != nil {
			return err
		}
	}

	print.InfoStatusEvent(os.Stdout, "Starting upgrade...")

	mtls, err := IsMTLSEnabled()
	if err != nil {
		return err
	}

	var vals map[string]interface{}
	var ca []byte
	var issuerCert []byte
	var issuerKey []byte

	if mtls {
		secret, sErr := getTrustChainSecret()
		if sErr != nil {
			return sErr
		}

		ca = secret.Data["ca.crt"]
		issuerCert = secret.Data["issuer.crt"]
		issuerKey = secret.Data["issuer.key"]
	}

	ha := highAvailabilityEnabled(status)
	vals, err = upgradeChartValues(string(ca), string(issuerCert), string(issuerKey), ha, mtls, conf)
	if err != nil {
		return err
	}

	if !isDowngrade(conf.RuntimeVersion, daprVersion) {
		err = applyCRDs("v" + conf.RuntimeVersion)
		if err != nil {
			return fmt.Errorf("unable to apply CRDs: %w", err)
		}
	} else {
		print.InfoStatusEvent(os.Stdout, "Downgrade detected, skipping CRDs.")
	}

	chart, err := GetDaprHelmChartName(helmConf)
	if err != nil {
		return err
	}

	// Deal with known race condition when applying both CRD and CR close together. The Helm upgrade fails
	// when a CR is applied tries to be applied before the CRD is fully registered. On each retry we need a
	// fresh client since the kube client locally caches the last OpenAPI schema it received from the server.
	// See https://github.com/kubernetes/kubectl/issues/1179
	_, err = helmUpgrade(upgradeClient, chart, controlPlaneChart, vals, WithRetry(5, 100*time.Millisecond))
	if err != nil {
		return fmt.Errorf("failure while running upgrade: %w", err)
	}

	if dashboardChart != nil {
		if dashboardExists {
			if _, err = upgradeClient.Run(dashboardReleaseName, dashboardChart, vals); err != nil {
				return err
			}
		} else {
			// We need to install Dashboard since it does not exist yet.
			err = install(dashboardReleaseName, conf.DashboardVersion, helmRepo, InitConfiguration{
				DashboardVersion: conf.DashboardVersion,
				Namespace:        upgradeClient.Namespace,
				Wait:             upgradeClient.Wait,
				Timeout:          conf.Timeout,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// WithRetry enables retry with the specified max retries and retry interval.
func WithRetry(maxRetries int, retryInterval time.Duration) UpgradeOption {
	return func(o *UpgradeOptions) {
		o.WithRetry = true
		o.MaxRetries = maxRetries
		o.RetryInterval = retryInterval
	}
}

func helmUpgrade(client *helm.Upgrade, name string, chart *chart.Chart, vals map[string]interface{}, options ...UpgradeOption) (*release.Release, error) {
	upgradeOptions := &UpgradeOptions{
		WithRetry:     false,
		MaxRetries:    0,
		RetryInterval: 0,
	}

	// Apply functional options.
	for _, option := range options {
		option(upgradeOptions)
	}

	var release *release.Release
	for attempt := 1; ; attempt++ {
		_, err := client.Run(name, chart, vals)
		if err == nil {
			// operation succeeded, no need to retry.
			break
		}

		if !upgradeOptions.WithRetry || attempt >= upgradeOptions.MaxRetries {
			// If not retrying or reached max retries, return the error.
			return nil, fmt.Errorf("max retries reached, unable to run command: %w", err)
		}

		print.PendingStatusEvent(os.Stdout, "Retrying after %s...", upgradeOptions.RetryInterval)
		time.Sleep(upgradeOptions.RetryInterval)

		// create a totally new helm client, this ensures that we fetch a fresh openapi schema from the server on each attempt.
		client, _, err = newUpgradeClient(client.Namespace, UpgradeConfig{
			Timeout: uint(client.Timeout), //nolint:gosec
		})
		if err != nil {
			return nil, fmt.Errorf("unable to create helm client: %w", err)
		}
	}

	return release, nil
}

func highAvailabilityEnabled(status []StatusOutput) bool {
	for _, s := range status {
		if s.Name == "dapr-dashboard" {
			continue
		}
		// Skip the scheduler server because it's in HA mode by default since version 1.15.0
		// This will fall back to other dapr services to determine if HA mode is enabled.
		if strings.HasPrefix(s.Name, "dapr-scheduler-server") {
			continue
		}
		if s.Replicas > 1 {
			return true
		}
	}
	return false
}

func applyCRDs(version string) error {
	for _, crd := range crds {
		url := fmt.Sprintf("https://raw.githubusercontent.com/dapr/dapr/%s/charts/dapr/crds/%s.yaml", version, crd)

		resp, _ := http.Get(url) //nolint:gosec
		if resp != nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()

			_, err := utils.RunCmdAndWait("kubectl", "apply", "-f", url)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func upgradeChartValues(ca, issuerCert, issuerKey string, haMode, mtls bool, conf UpgradeConfig) (map[string]interface{}, error) {
	chartVals := map[string]interface{}{}
	globalVals := conf.Args
	err := utils.ValidateImageVariant(conf.ImageVariant)
	if err != nil {
		return nil, err
	}
	globalVals = append(globalVals, "global.tag="+utils.GetVariantVersion(conf.RuntimeVersion, conf.ImageVariant))

	if mtls && ca != "" && issuerCert != "" && issuerKey != "" {
		globalVals = append(globalVals, "dapr_sentry.tls.root.certPEM="+ca,
			"dapr_sentry.tls.issuer.certPEM="+issuerCert,
			"dapr_sentry.tls.issuer.keyPEM="+issuerKey,
		)
	} else {
		globalVals = append(globalVals, "global.mtls.enabled=false")
	}
	if len(conf.ImageRegistryURI) != 0 {
		globalVals = append(globalVals, "global.registry="+conf.ImageRegistryURI)
	}
	if haMode {
		globalVals = append(globalVals, "global.ha.enabled=true")
	}

	for _, v := range globalVals {
		if err := strvals.ParseInto(v, chartVals); err != nil {
			return nil, err
		}
	}
	return chartVals, nil
}

func isDowngrade(targetVersion, existingVersion string) bool {
	target, _ := version.NewVersion(targetVersion)
	existing, err := version.NewVersion(existingVersion)
	if err != nil {
		print.FailureStatusEvent(
			os.Stderr,
			fmt.Sprintf("Upgrade failed, %s. The current installed version does not have sematic versioning", err.Error()))
		os.Exit(1)
	}
	return target.LessThan(existing)
}

func newUpgradeClient(namespace string, cfg UpgradeConfig) (*helm.Upgrade, *helm.Configuration, error) {
	helmCfg, err := helmConfig(namespace)
	if err != nil {
		return nil, nil, err
	}

	client := helm.NewUpgrade(helmCfg)
	client.ResetValues = true
	client.Namespace = namespace
	client.CleanupOnFail = true
	client.Wait = true
	client.Timeout = time.Duration(cfg.Timeout) * time.Second //nolint:gosec

	return client, helmCfg, nil
}
