package kubernetes

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/dapr/dapr/pkg/apis/configuration/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	if err != nil {
		return nil, err
	}

	for _, c := range configs.Items {
		if c.GetName() == systemConfigName {
			return &c, nil
		}
	}
	return nil, errors.New("system configuration not found")
}

// ExportTrustChain takes the root cert, issuer cert and issuer key from a k8s cluster and saves them in a given directory
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

func getTrustChainSecret() (*corev1.Secret, error) {
	_, client, err := GetKubeConfigClient()
	if err != nil {
		return nil, err
	}

	c, err := getSystemConfig()
	if err != nil {
		return nil, err
	}
	res, err := client.CoreV1().Secrets(c.GetNamespace()).List(meta_v1.ListOptions{})
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

// Expiry returns the expiry time for the root cert
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
