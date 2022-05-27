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
	"os"
	"time"

	helm "helm.sh/helm/v3/pkg/action"
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
}

var crdsFullResources = []string{
	"components.dapr.io",
	"configurations.dapr.io",
	"subscriptions.dapr.io",
}

type UpgradeConfig struct {
	RuntimeVersion   string
	Args             []string
	Timeout          uint
	ImageRegistryURI string
}

func Upgrade(conf UpgradeConfig) error {
	status, err := GetDaprResourcesStatus()
	if err != nil {
		return err
	}

	daprVersion := GetDaprVersion(status)
	print.InfoStatusEvent(os.Stdout, "Dapr control plane version %s detected in namespace %s", daprVersion, status[0].Namespace)

	helmConf, err := helmConfig(status[0].Namespace)
	if err != nil {
		return err
	}

	daprChart, err := daprChart(conf.RuntimeVersion, helmConf)
	if err != nil {
		return err
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

	if _, err = upgradeClient.Run(chart, daprChart, vals); err != nil {
		return err
	}
	return nil
}

func highAvailabilityEnabled(status []StatusOutput) bool {
	for _, s := range status {
		if s.Replicas > 1 {
			return true
		}
	}
	return false
}

func applyCRDs(version string) error {
	for _, crd := range crds {
		url := fmt.Sprintf("https://raw.githubusercontent.com/dapr/dapr/%s/charts/dapr/crds/%s.yaml", version, crd)
		_, err := utils.RunCmdAndWait("kubectl", "apply", "-f", url)
		if err != nil {
			return err
		}
	}
	return nil
}

func upgradeChartValues(ca, issuerCert, issuerKey string, haMode, mtls bool, conf UpgradeConfig) (map[string]interface{}, error) {
	chartVals := map[string]interface{}{}
	globalVals := conf.Args

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
