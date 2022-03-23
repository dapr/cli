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
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	helm "helm.sh/helm/v3/pkg/action"
	"k8s.io/helm/pkg/strvals"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/dapr/pkg/sentry/certs"
	"github.com/dapr/dapr/pkg/sentry/csr"
)

type RenewCertificateParams struct {
	RootCertificateFilePath   string
	IssuerCertificateFilePath string
	IssuerPrivateKeyFilePath  string
	RootPrivateKeyFilePath    string
	ValidUntil                time.Duration
	Timeout                   uint
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
	err = renewCertificate(rootCertBytes, issuerCertBytes, issuerKeyBytes, conf.Timeout)
	if err != nil {
		return err
	}
	return nil
}

func parseCertificateFiles(rootCert, issuerCert, issuerKey string) ([]byte, []byte, []byte, error) {
	rootCertBytes, err := ioutil.ReadFile(rootCert)
	if err != nil {
		return nil, nil, nil, err
	}
	issuerCertBytes, err := ioutil.ReadFile(issuerCert)
	if err != nil {
		return nil, nil, nil, err
	}
	issuerKeyBytes, err := ioutil.ReadFile(issuerKey)
	if err != nil {
		return nil, nil, nil, err
	}
	return rootCertBytes, issuerCertBytes, issuerKeyBytes, nil
}

func renewCertificate(rootCert, issuerCert, issuerKey []byte, timeout uint) error {
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
	upgradeClient.Timeout = time.Duration(timeout) * time.Second
	upgradeClient.Namespace = status[0].Namespace

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
		privateKeyBytes, err := ioutil.ReadFile(privateKeyFile)
		if err != nil {
			return nil, nil, nil, err
		}
		rootKey, err = x509.ParseECPrivateKey(privateKeyBytes)
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
	rootCsr, err := csr.GenerateRootCertCSR("dapr.io/sentry", "cluster.local", &rootKey.PublicKey, validUntil, time.Minute*15)
	if err != nil {
		return nil, nil, nil, err
	}
	rootCertBytes, err := x509.CreateCertificate(rand.Reader, rootCsr, rootCsr, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, nil, err
	}
	rootCertPem := pem.EncodeToMemory(&pem.Block{Type: certs.Certificate, Bytes: rootCertBytes})
	rootCert, err := x509.ParseCertificate(rootCertBytes)
	if err != nil {
		return nil, nil, nil, err
	}

	issuerKey, err := certs.GenerateECPrivateKey()
	if err != nil {
		return nil, nil, nil, err
	}
	encodedKey, err := x509.MarshalECPrivateKey(issuerKey)
	if err != nil {
		return nil, nil, nil, err
	}
	issuerKeyPem := pem.EncodeToMemory(&pem.Block{Type: certs.ECPrivateKey, Bytes: encodedKey})

	issuerCsr, err := csr.GenerateIssuerCertCSR("cluster.local", &issuerKey.PublicKey, validUntil, time.Minute*15)
	if err != nil {
		return nil, nil, nil, err
	}

	issuerCertBytes, err := x509.CreateCertificate(rand.Reader, issuerCsr, rootCert, &issuerKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, nil, err
	}
	issuerCertPem := pem.EncodeToMemory(&pem.Block{Type: certs.Certificate, Bytes: issuerCertBytes})

	return rootCertPem, issuerCertPem, issuerKeyPem, nil
}
