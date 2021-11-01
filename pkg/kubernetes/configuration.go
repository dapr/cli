package kubernetes

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/dapr/dapr/pkg/apis/configuration/v1alpha1"
)

func GetDefaultConfiguration() v1alpha1.Configuration {
	return v1alpha1.Configuration{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "daprsystem",
		},
		Spec: v1alpha1.ConfigurationSpec{
			MTLSSpec: v1alpha1.MTLSSpec{
				Enabled:          true,
				WorkloadCertTTL:  "24h",
				AllowedClockSkew: "15m",
			},
		},
	}
}
