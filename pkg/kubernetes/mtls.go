package kubernetes

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultConfigName = "default"
)

func IsMTLSEnabled() (bool, error) {
	client, err := DaprClient()
	if err != nil {
		return false, err
	}

	configs, err := client.ConfigurationV1alpha1().Configurations(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, c := range configs.Items {
		if c.GetName() == defaultConfigName {
			return c.Spec.MTLSSpec.Enabled, nil
		}
	}
	return false, nil
}
