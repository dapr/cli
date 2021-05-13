// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
	helm "helm.sh/helm/v3/pkg/action"
	"k8s.io/helm/pkg/strvals"
)

const operatorName = "dapr-operator"

var crds = []string{
	"components",
	"configuration",
	"subscription",
}

type UpgradeConfig struct {
	RuntimeVersion string
	Args           []string
	Timeout        uint
}

func Upgrade(conf UpgradeConfig) error {
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

	var daprVersion string
	for _, s := range status {
		if s.Name == operatorName {
			daprVersion = s.Version
		}
	}
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
	upgradeClient.ReuseValues = true
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
	vals, err = upgradeChartValues(string(ca), string(issuerCert), string(issuerKey), ha, conf.Args)
	if err != nil {
		return err
	}

	err = applyCRDs(fmt.Sprintf("v%s", conf.RuntimeVersion))
	if err != nil {
		return err
	}

	listClient := helm.NewList(helmConf)
	releases, err := listClient.Run()
	if err != nil {
		return err
	}

	var chart string
	for _, r := range releases {
		if r.Chart != nil && strings.Contains(r.Chart.Name(), "dapr") {
			chart = r.Name
			break
		}
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

func upgradeChartValues(ca, issuerCert, issuerKey string, haMode bool, args []string) (map[string]interface{}, error) {
	chartVals := map[string]interface{}{}
	globalVals := args

	if ca != "" && issuerCert != "" && issuerKey != "" {
		globalVals = append(globalVals, fmt.Sprintf("dapr_sentry.tls.root.certPEM=%s", ca),
			fmt.Sprintf("dapr_sentry.tls.issuer.certPEM=%s", issuerCert),
			fmt.Sprintf("dapr_sentry.tls.issuer.keyPEM=%s", issuerKey),
		)
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
