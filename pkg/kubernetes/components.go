// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"io"
	"os"
	"strings"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dapr/cli/pkg/age"
	"github.com/dapr/cli/utils"
	v1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
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

// PrintComponents prints all Dapr components.
func PrintComponents(name, outputFormat string) error {
	return writeComponents(os.Stdout, func() (*v1alpha1.ComponentList, error) {
		client, err := DaprClient()
		if err != nil {
			return nil, err
		}
		//nolint
		return client.ComponentsV1alpha1().Components(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
	}, name, outputFormat)
}

//nolint
func writeComponents(writer io.Writer, getConfigFunc func() (*v1alpha1.ComponentList, error), name, outputFormat string) error {
	confs, err := getConfigFunc()
	if err != nil {
		return err
	}

	filtered := []v1alpha1.Component{}
	filteredSpecs := []configurationDetailedOutput{}
	for _, c := range confs.Items {
		confName := c.GetName()
		if confName == "daprsystem" {
			continue
		}

		if name == "" || strings.EqualFold(confName, name) {
			filtered = append(filtered, c)
			filteredSpecs = append(filteredSpecs, configurationDetailedOutput{
				Name: confName,
				Spec: c.Spec,
			})
		}
	}

	if outputFormat == "" || outputFormat == "list" {
		return printComponentList(writer, filtered)
	}
	//nolint
	return utils.PrintDetail(writer, outputFormat, filteredSpecs)
}

func printComponentList(writer io.Writer, list []v1alpha1.Component) error {
	co := []ComponentsOutput{}
	for _, c := range list {
		co = append(co, ComponentsOutput{
			Name:    c.GetName(),
			Type:    c.Spec.Type,
			Created: c.CreationTimestamp.Format("2006-01-02 15:04.05"),
			Age:     age.GetAge(c.CreationTimestamp.Time),
			Version: c.Spec.Version,
			Scopes:  strings.Join(c.Scopes, ","),
		})
	}
	//nolint
	return utils.MarshalAndWriteTable(writer, co)
}
