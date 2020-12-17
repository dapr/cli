// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"errors"
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
	helm "helm.sh/helm/v3/pkg/action"
	"k8s.io/helm/pkg/strvals"
)

var crds = []string{
	"components",
	"configuration",
	"subscription",
}

type UpgradeConfig struct {
	RuntimeVersion string
}

func Upgrade(conf UpgradeConfig) error {
	helmConf, err := helmConfig("dapr-system")
	if err != nil {
		return err
	}

	daprChart, err := daprChart(conf.RuntimeVersion, helmConf)
	if err != nil {
		return err
	}

	sc, err := NewStatusClient()
	if err != nil {
		return err
	}

	status, err := sc.Status()
	if err != nil {
		return err
	}

	if len(status) == 0 {
		return errors.New("dapr is not installed in your cluster")
	}

	print.InfoStatusEvent(os.Stdout, "Dapr control plane version %s detected", status[0].Version)

	upgradeClient := helm.NewUpgrade(helmConf)
	upgradeClient.ResetValues = true
	upgradeClient.Namespace = status[0].Namespace
	upgradeClient.CleanupOnFail = true
	upgradeClient.Wait = true

	print.InfoStatusEvent(os.Stdout, "Starting upgrade...")

	mtls, err := IsMTLSEnabled()
	if err != nil {
		return err
	}

	var vals map[string]interface{}

	if mtls {
		secret, sErr := getTrustChainSecret()
		if sErr != nil {
			return sErr
		}

		ca := secret.Data["ca.crt"]
		issuerCert := secret.Data["issuer.crt"]
		issuerKey := secret.Data["issuer.key"]

		vals, err = mtlsChartValues(string(ca), string(issuerCert), string(issuerKey))
		if err != nil {
			return err
		}
	}

	err = applyCRDs(fmt.Sprintf("v%s", conf.RuntimeVersion))
	if err != nil {
		return err
	}

	if _, err = upgradeClient.Run("dapr", daprChart, vals); err != nil {
		return err
	}
	return nil
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

func mtlsChartValues(ca, issuerCert, issuerKey string) (map[string]interface{}, error) {
	chartVals := map[string]interface{}{}
	globalVals := []string{
		fmt.Sprintf("dapr_sentry.tls.root.certPEM=%s", ca),
		fmt.Sprintf("dapr_sentry.tls.issuer.certPEM=%s", issuerCert),
		fmt.Sprintf("dapr_sentry.tls.issuer.keyPEM=%s", issuerKey),
	}

	for _, v := range globalVals {
		if err := strvals.ParseInto(v, chartVals); err != nil {
			return nil, err
		}
	}
	return chartVals, nil
}
