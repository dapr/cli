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
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/dapr/pkg/apis/configuration/v1alpha1"
)

const (
	systemConfigName         = "daprsystem"
	trustBundleSecretName    = "dapr-trust-bundle" // nolint:gosec
	warningDaysForCertExpiry = 30                  // in days
)

func IsMTLSEnabled() (bool, error) {
	c, err := getSystemConfig()
	if err != nil {
		return false, err
	}
	return c.Spec.MTLSSpec.Enabled, nil
}

func getSystemConfig() (*v1alpha1.Configuration, error) {
	client, err := DaprClient()
	if err != nil {
		return nil, err
	}

	configs, err := client.ConfigurationV1alpha1().Configurations(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
	// This means that the Dapr Configurations CRD is not installed and
	// therefore no configuration items exist.
	if apierrors.IsNotFound(err) {
		configs = &v1alpha1.ConfigurationList{
			Items: []v1alpha1.Configuration{},
		}
	} else if err != nil {
		return nil, err
	}

	for _, c := range configs.Items {
		if c.GetName() == systemConfigName {
			return &c, nil
		}
	}

	return nil, errors.New("system configuration not found")
}

// ExportTrustChain takes the root cert, issuer cert and issuer key from a k8s cluster and saves them in a given directory.
func ExportTrustChain(outputDir string) error {
	_, err := os.Stat(outputDir)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(outputDir, 0755)
		if errDir != nil {
			return err
		}
	}

	secret, err := getTrustChainSecret()
	if err != nil {
		return err
	}

	ca := secret.Data["ca.crt"]
	issuerCert := secret.Data["issuer.crt"]
	issuerKey := secret.Data["issuer.key"]

	err = ioutil.WriteFile(filepath.Join(outputDir, "ca.crt"), ca, 0600)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(outputDir, "issuer.crt"), issuerCert, 0600)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(outputDir, "issuer.key"), issuerKey, 0600)
	if err != nil {
		return err
	}
	return nil
}

// Check and warn if cert expiry is less than `warningDaysForCertExpiry` days.
func CheckForCertExpiry() {
	expiry, err := Expiry()
	// The intent is to warn for certificate expiry only when it can be fetched.
	// Do not show any kind of errors with normal command flow.
	if err == nil {
		daysRemaining := int(expiry.Sub(time.Now().UTC()).Hours() / 24)
		if daysRemaining < warningDaysForCertExpiry {
			warningMessage := ""
			if daysRemaining == 0 {
				warningMessage = "Root certificate of your kubernetes cluster expires today"
			} else if daysRemaining < 0 {
				warningMessage = "Root certificate your kubernetes cluster already expired"
			} else {
				warningMessage = fmt.Sprintf("Root certificate your kubernetes cluster expires in %v days", daysRemaining)
			}
			color.Set(color.FgHiYellow)
			helpMessage := "Kindly renew to avoid any service interuptions."
			message := fmt.Sprintf("%s. Expiry date: %s. \n %s", warningMessage, expiry.Format(time.RFC1123), helpMessage)
			print.WarningStatusEvent(os.Stdout, message)
			color.Unset()
		}
	}
}

func getTrustChainSecret() (*corev1.Secret, error) {
	_, client, err := GetKubeConfigClient()
	if err != nil {
		return nil, err
	}

	c, err := getSystemConfig()
	if err != nil {
		return nil, err
	}
	res, err := client.CoreV1().Secrets(c.GetNamespace()).List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, i := range res.Items {
		if i.GetName() == trustBundleSecretName {
			return &i, nil
		}
	}
	return nil, fmt.Errorf("could not find trust chain secret named %s in namespace %s", trustBundleSecretName, c.GetNamespace())
}

// Expiry returns the expiry time for the root cert.
func Expiry() (*time.Time, error) {
	secret, err := getTrustChainSecret()
	if err != nil {
		return nil, err
	}

	caCrt := secret.Data["ca.crt"]
	block, _ := pem.Decode(caCrt)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return &cert.NotAfter, nil
}
