package kubernetes

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/dapr/dapr/pkg/apis/configuration/v1alpha1"
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

	_, client, err := GetKubeConfigClient()
	if err != nil {
		return err
	}

	c, err := getSystemConfig()
	if err != nil {
		return err
	}
	res, err := client.CoreV1().Secrets(c.GetNamespace()).List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}

	for _, i := range res.Items {
		if i.GetName() == trustBundleSecretName {
			ca := i.Data["ca.crt"]
			issuerCert := i.Data["issuer.crt"]
			issuerKey := i.Data["issuer.key"]

			err := ioutil.WriteFile(filepath.Join(outputDir, "ca.crt"), ca, 0600)
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
			break
		}
	}
	return nil
}
