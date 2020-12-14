// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"encoding/json"
	"io"
	"os"
	"strconv"

	"github.com/dapr/cli/pkg/age"
	"github.com/dapr/cli/utils"
	v1alpha1 "github.com/dapr/dapr/pkg/apis/configuration/v1alpha1"
	"github.com/gocarina/gocsv"
	"gopkg.in/yaml.v2"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type configurationsOutput struct {
	Name           string `csv:"Name"`
	TracingEnabled bool   `csv:"TRACING-ENABLED"`
	MetricsEnabled bool   `csv:"METRICS-ENABLED"`
	Age            string `csv:"AGE"`
	Created        string `csv:"CREATED"`
}

type configurationDetailedOutput struct {
	Name string      `json:"name" yaml:"name"`
	Spec interface{} `json:"spec" yaml:"spec"`
}

// PrintConfigurations prints all Dapr configurations.
func PrintConfigurations(name, outputFormat string) error {
	return writeConfigurations(os.Stdout, func() (*v1alpha1.ConfigurationList, error) {
		client, err := DaprClient()
		if err != nil {
			return nil, err
		}

		return client.ConfigurationV1alpha1().Configurations(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
	}, name, outputFormat)
}

func writeConfigurations(writer io.Writer, getConfigFunc func() (*v1alpha1.ConfigurationList, error), name, outputFormat string) error {
	confs, err := getConfigFunc()
	if err != nil {
		return err
	}

	filtered := []v1alpha1.Configuration{}
	filteredSpecs := []configurationDetailedOutput{}
	for _, c := range confs.Items {
		confName := c.GetName()
		if confName == "daprsystem" {
			continue
		}

		if name == "" || confName == name {
			filtered = append(filtered, c)
			filteredSpecs = append(filteredSpecs, configurationDetailedOutput{
				Name: confName,
				Spec: c.Spec,
			})
		}
	}

	if outputFormat == "" || outputFormat == "list" {
		return printList(writer, filtered)
	}

	return printDetail(writer, outputFormat, filteredSpecs)
}

func printDetail(writer io.Writer, outputFormat string, list []configurationDetailedOutput) error {
	var err error
	output := []byte{}
	var obj interface{} = list
	if len(list) == 1 {
		obj = list[0]
	}
	if outputFormat == "yaml" {
		output, err = yaml.Marshal(obj)
	}

	if outputFormat == "json" {
		output, err = json.MarshalIndent(obj, "", "  ")
	}

	if err != nil {
		return err
	}

	writer.Write(output)
	return nil
}

func printList(writer io.Writer, list []v1alpha1.Configuration) error {
	co := []configurationsOutput{}
	for _, c := range list {
		co = append(co, configurationsOutput{
			TracingEnabled: tracingEnabled(c.Spec.TracingSpec),
			Name:           c.GetName(),
			MetricsEnabled: c.Spec.MetricSpec.Enabled,
			Created:        c.CreationTimestamp.Format("2006-01-02 15:04.05"),
			Age:            age.GetAge(c.CreationTimestamp.Time),
		})
	}

	table, err := gocsv.MarshalString(co)
	if err != nil {
		return err
	}

	utils.WriteTable(writer, table)
	return nil
}

func tracingEnabled(spec v1alpha1.TracingSpec) bool {
	sr, err := strconv.ParseFloat(spec.SamplingRate, 32)
	if err != nil {
		return false
	}
	return sr > 0
}
