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
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	helm "helm.sh/helm/v3/pkg/action"
	"k8s.io/helm/pkg/strvals"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
	"github.com/dapr/dapr/pkg/sentry/ca"
	"github.com/dapr/dapr/pkg/sentry/certs"
)

type RenewCertificateParams struct {
	RootCertificateFilePath   string
	IssuerCertificateFilePath string
	IssuerPrivateKeyFilePath  string
	RootPrivateKeyFilePath    string
	ValidUntil                time.Duration
	Timeout                   uint
	ImageVariant              string
}

func RenewCertificate(conf RenewCertificateParams) error {
	var rootCertBytes []byte
	var issuerCertBytes []byte
	var issuerKeyBytes []byte
	var err error
	if conf.RootCertificateFilePath != "" && conf.IssuerCertificateFilePath != "" && conf.IssuerPrivateKeyFilePath != "" {
		rootCertBytes, issuerCertBytes, issuerKeyBytes, err = parseCertificateFiles(
			conf.RootCertificateFilePath,
			conf.IssuerCertificateFilePath,
			conf.IssuerPrivateKeyFilePath)

		if err != nil {
			return err
		}
	} else {
		rootCertBytes, issuerCertBytes, issuerKeyBytes, err = GenerateNewCertificates(
			conf.ValidUntil,
			conf.RootPrivateKeyFilePath)

		if err != nil {
			return err
		}
	}
	print.InfoStatusEvent(os.Stdout, "Updating certifcates in your Kubernetes cluster")
	err = renewCertificate(rootCertBytes, issuerCertBytes, issuerKeyBytes, conf.Timeout, conf.ImageVariant)
	if err != nil {
		return err
	}
	return nil
}

func parseCertificateFiles(rootCert, issuerCert, issuerKey string) ([]byte, []byte, []byte, error) {
	rootCertBytes, err := os.ReadFile(rootCert)
	if err != nil {
		return nil, nil, nil, err
	}
	issuerCertBytes, err := os.ReadFile(issuerCert)
	if err != nil {
		return nil, nil, nil, err
	}
	issuerKeyBytes, err := os.ReadFile(issuerKey)
	if err != nil {
		return nil, nil, nil, err
	}
	return rootCertBytes, issuerCertBytes, issuerKeyBytes, nil
}

func renewCertificate(rootCert, issuerCert, issuerKey []byte, timeout uint, imageVariant string) error {
	var daprVersion, daprImageVariant string
	status, err := GetDaprResourcesStatus()
	if err != nil {
		return err
	}
	daprVersion = GetDaprVersion(status)
	print.InfoStatusEvent(os.Stdout, "Dapr control plane version %s detected in namespace %s", daprVersion, status[0].Namespace)

	// Get the control plane version from daprversion(1.x.x-mariner), if image variant is provided.
	// Here, imageVariant is used only to extract the actual control plane version,
	// and do some validation on top of that.
	if imageVariant != "" {
		daprVersion, daprImageVariant = utils.GetVersionAndImageVariant(daprVersion)
		if daprImageVariant != imageVariant {
			return fmt.Errorf("error in parsing dapr version. found image variant %q is not same as provided value %q", daprImageVariant, imageVariant)
		}
	}

	helmConf, err := helmConfig(status[0].Namespace)
	if err != nil {
		return err
	}

	daprChart, err := daprChart(daprVersion, "dapr", helmConf)
	if err != nil {
		return err
	}
	upgradeClient := helm.NewUpgrade(helmConf)

	// Reuse the existing helm configuration values i.e. tags, registry, etc.
	upgradeClient.ReuseValues = true
	upgradeClient.Wait = true
	upgradeClient.Timeout = time.Duration(timeout) * time.Second
	upgradeClient.Namespace = status[0].Namespace

	// Override the helm configuration values with the new certificates.
	vals, err := createHelmParamsForNewCertificates(string(rootCert), string(issuerCert), string(issuerKey))
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

func createHelmParamsForNewCertificates(ca, issuerCert, issuerKey string) (map[string]interface{}, error) {
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

func GenerateNewCertificates(validUntil time.Duration, privateKeyFile string) ([]byte, []byte, []byte, error) {
	var rootKey *ecdsa.PrivateKey
	if privateKeyFile != "" {
		privateKeyBytes, err := os.ReadFile(privateKeyFile)
		if err != nil {
			return nil, nil, nil, err
		}
		privateKeyPemBlock, _ := pem.Decode(privateKeyBytes)
		if privateKeyPemBlock == nil {
			return nil, nil, nil, errors.New("provided private key file is not pem encoded")
		}
		rootKey, err = x509.ParseECPrivateKey(privateKeyPemBlock.Bytes)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		var err error
		rootKey, err = certs.GenerateECPrivateKey()
		if err != nil {
			return nil, nil, nil, err
		}
	}
	systemConfig, err := GetDaprControlPlaneCurrentConfig()
	if err != nil {
		return nil, nil, nil, err
	}
	allowedClockSkew, err := time.ParseDuration(systemConfig.Spec.MTLSSpec.AllowedClockSkew)
	if err != nil {
		return nil, nil, nil, err
	}
	_, rootCertPem, issuerCertPem, issuerKeyPem, err := ca.GetNewSelfSignedCertificates(rootKey, validUntil, allowedClockSkew)
	if err != nil {
		return nil, nil, nil, err
	}
	return rootCertPem, issuerCertPem, issuerKeyPem, nil
}
