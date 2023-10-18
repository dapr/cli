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
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/go-version"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/strvals"

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

func Upgrade(ctx context.Context, conf UpgradeConfig) error {
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

	helmConf, err := helmConfig(status[0].Namespace)
	if err != nil {
		return err
	}

	controlPlaneChart, err := getHelmChart(conf.RuntimeVersion, "dapr", helmRepo, helmConf)
	if err != nil {
		return err
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
		uninstallClient.Timeout = time.Duration(conf.Timeout) * time.Second

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

	upgradeClient := helm.NewUpgrade(helmConf)
	upgradeClient.ResetValues = true
	upgradeClient.Namespace = status[0].Namespace
	upgradeClient.CleanupOnFail = true
	upgradeClient.Wait = true
	upgradeClient.Timeout = time.Duration(conf.Timeout) * time.Second

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
		err = applyCRDs(fmt.Sprintf("v%s", conf.RuntimeVersion))
		if err != nil {
			return err
		}
	} else {
		print.InfoStatusEvent(os.Stdout, "Downgrade detected, skipping CRDs.")
	}

	chart, err := GetDaprHelmChartName(helmConf)
	if err != nil {
		return err
	}

	client, err := Client()
	if err != nil {
		return err
	}

	var mutatingWebhookConf *admissionregistrationv1.MutatingWebhookConfiguration
	if is12to11Downgrade(conf.RuntimeVersion, daprVersion) {
		print.InfoStatusEvent(os.Stdout, "Downgrade from 1.12 to 1.11 detected, temporarily deleting injector mutating webhook...")

		mutatingWebhookConf, err = client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, "dapr-sidecar-injector", metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		err = client.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(ctx, "dapr-sidecar-injector", metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	if _, err = upgradeClient.Run(chart, controlPlaneChart, vals); err != nil {
		if mutatingWebhookConf != nil {
			_, merr := client.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(ctx, mutatingWebhookConf, metav1.CreateOptions{})
			return errors.Join(err, merr)
		}
		return err
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

func highAvailabilityEnabled(status []StatusOutput) bool {
	for _, s := range status {
		if s.Name == "dapr-dashboard" {
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
		if resp != nil && resp.StatusCode == 200 {
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
	globalVals = append(globalVals, fmt.Sprintf("global.tag=%s", utils.GetVariantVersion(conf.RuntimeVersion, conf.ImageVariant)))

	if mtls && ca != "" && issuerCert != "" && issuerKey != "" {
		globalVals = append(globalVals, fmt.Sprintf("dapr_sentry.tls.root.certPEM=%s", ca),
			fmt.Sprintf("dapr_sentry.tls.issuer.certPEM=%s", issuerCert),
			fmt.Sprintf("dapr_sentry.tls.issuer.keyPEM=%s", issuerKey),
		)
	} else {
		globalVals = append(globalVals, "global.mtls.enabled=false")
	}
	if len(conf.ImageRegistryURI) != 0 {
		globalVals = append(globalVals, fmt.Sprintf("global.registry=%s", conf.ImageRegistryURI))
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

func is12to11Downgrade(targetVersion, existingVersion string) bool {
	target, _ := version.NewVersion(targetVersion)
	existing, _ := version.NewVersion(existingVersion)
	if target == nil || existing == nil {
		return false
	}

	tset := target.Segments()
	eset := existing.Segments()

	if len(eset) < 2 || len(tset) < 2 {
		return false
	}

	return eset[0] == 1 && eset[1] == 12 && tset[0] == 1 && tset[1] == 11
}
