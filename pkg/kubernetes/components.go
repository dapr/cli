// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"strings"

	"github.com/dapr/cli/pkg/age"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComponentsOutput represent a Dapr component.
type ComponentsOutput struct {
	Name    string `csv:"Name"`
	Type    string `csv:"Type"`
	Version string `csv:"VERSION"`
	Scopes  string `csv:"SCOPES"`
	Created string `csv:"CREATED"`
	Age     string `csv:"AGE"`
}

// List outputs all Dapr components.
func Components() ([]ComponentsOutput, error) {
	client, err := DaprClient()
	if err != nil {
		return nil, err
	}

	comps, err := client.ComponentsV1alpha1().Components(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	co := []ComponentsOutput{}
	for _, c := range comps.Items {
		co = append(co, ComponentsOutput{
			Name:    c.GetName(),
			Type:    c.Spec.Type,
			Created: c.CreationTimestamp.Format("2006-01-02 15:04.05"),
			Age:     age.GetAge(c.CreationTimestamp.Time),
			Version: c.Spec.Version,
			Scopes:  strings.Join(c.Scopes, ","),
		})
	}
	return co, nil
}
