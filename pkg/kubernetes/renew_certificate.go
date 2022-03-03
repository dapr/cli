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
	"io/ioutil"
	"os"

	"github.com/dapr/cli/pkg/print"
	helm "helm.sh/helm/v3/pkg/action"
	"k8s.io/helm/pkg/strvals"
)

func RenewCertificate(caRootCertificateFile, issuerPrivateKeyFile, issuerPublicCertificateFile string) error {
	ca, err := ioutil.ReadFile(caRootCertificateFile)
	if err != nil {
		return err
	}
	issuerCert, err := ioutil.ReadFile(issuerPublicCertificateFile)
	if err != nil {
		return err
	}
	issuerKey, err := ioutil.ReadFile(issuerPrivateKeyFile)
	if err != nil {
		return err
	}

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

	daprChart, err := daprChart(daprVersion, helmConf)
	if err != nil {
		return err
	}
	upgradeClient := helm.NewUpgrade(helmConf)
	upgradeClient.ReuseValues = true
	upgradeClient.Wait = true
	upgradeClient.Namespace = status[0].Namespace

	vals, err := setCertificateValues(string(ca), string(issuerCert), string(issuerKey))
	if err != nil {
		return err
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

func setCertificateValues(ca, issuerCert, issuerKey string) (map[string]interface{}, error) {
	chartVals := map[string]interface{}{}
	args := []string{}

	if ca != "" && issuerCert != "" && issuerKey != "" {
		args = append(args, fmt.Sprintf("dapr_sentry.tls.root.certPEM=%s", ca),
			fmt.Sprintf("dapr_sentry.tls.issuer.certPEM=%s", issuerCert),
			fmt.Sprintf("dapr_sentry.tls.issuer.keyPEM=%s", issuerKey),
		)
	} else {
		return nil, fmt.Errorf("parameters not found")
	}

	for _, v := range args {
		if err := strvals.ParseInto(v, chartVals); err != nil {
			return nil, err
		}
	}
	return chartVals, nil
}
