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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dapr/dapr/pkg/apis/configuration/v1alpha1"
)

const (
	systemConfigName      = "daprsystem"
	trustBundleSecretName = "dapr-trust-bundle" // nolint:gosec
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
		return nil, fmt.Errorf("error: %w", err)
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
			return fmt.Errorf("error: %w", err)
		}
	}

	secret, err := getTrustChainSecret()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	ca := secret.Data["ca.crt"]
	issuerCert := secret.Data["issuer.crt"]
	issuerKey := secret.Data["issuer.key"]

	err = ioutil.WriteFile(filepath.Join(outputDir, "ca.crt"), ca, 0600)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	err = ioutil.WriteFile(filepath.Join(outputDir, "issuer.crt"), issuerCert, 0600)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	err = ioutil.WriteFile(filepath.Join(outputDir, "issuer.key"), issuerKey, 0600)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}
	return nil
}

func getTrustChainSecret() (*corev1.Secret, error) {
	_, client, err := GetKubeConfigClient()
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	c, err := getSystemConfig()
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}
	res, err := client.CoreV1().Secrets(c.GetNamespace()).List(context.TODO(), meta_v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
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
		return nil, fmt.Errorf("error: %w", err)
	}

	caCrt := secret.Data["ca.crt"]
	block, _ := pem.Decode(caCrt)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}
	return &cert.NotAfter, nil
}
