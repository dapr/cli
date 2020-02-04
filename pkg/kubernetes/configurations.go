// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"github.com/dapr/cli/pkg/age"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComponentsOutput represent a Dapr component.
type ConfigurtionsOutput struct {
	Name            string `csv:"Name"`
	TracingEnabled  bool   `csv:"TRACING-ENABLED"`
	MTLSEnabled     bool   `csv:"MTLS-ENABLED"`
	WorkloadCertTTL string `csv:"MTLS-WORKLOAD-TTL"`
	ClockSkew       string `csv:"MTLS-CLOCK-SKEW"`
	Age             string `csv:"AGE"`
	Created         string `csv:"CREATED"`
}

// List outputs all Dapr configurations.
func Configurations() ([]ConfigurtionsOutput, error) {
	client, err := DaprClient()
	if err != nil {
		return nil, err
	}

	confs, err := client.ConfigurationV1alpha1().Configurations(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	co := []ConfigurtionsOutput{}
	for _, c := range confs.Items {
		co = append(co, ConfigurtionsOutput{
			Name:            c.GetName(),
			TracingEnabled:  c.Spec.TracingSpec.Enabled,
			MTLSEnabled:     c.Spec.MTLSSpec.Enabled,
			WorkloadCertTTL: c.Spec.MTLSSpec.WorkloadCertTTL,
			ClockSkew:       c.Spec.MTLSSpec.AllowedClockSkew,
			Created:         c.CreationTimestamp.Format("2006-01-02 15:04.05"),
			Age:             age.GetAge(c.CreationTimestamp.Time),
		})
	}
	return co, nil
}
