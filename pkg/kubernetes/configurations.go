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
	"io"
	"os"
	"strconv"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dapr/cli/pkg/age"
	"github.com/dapr/cli/utils"
	v1alpha1 "github.com/dapr/dapr/pkg/apis/configuration/v1alpha1"
)

type configurationsOutput struct {
	Namespace      string `csv:"Namespace"`
	Name           string `csv:"Name"`
	TracingEnabled bool   `csv:"TRACING-ENABLED"`
	MetricsEnabled bool   `csv:"METRICS-ENABLED"`
	Age            string `csv:"AGE"`
	Created        string `csv:"CREATED"`
}

type configurationDetailedOutput struct {
	Name      string      `json:"name" yaml:"name"`
	Namespace string      `json:"namespace" yaml:"namespace"`
	Spec      interface{} `json:"spec" yaml:"spec"`
}

// PrintConfigurations prints all Dapr configurations.
func PrintConfigurations(name, namespace, outputFormat string) error {
	return writeConfigurations(os.Stdout, func() (*v1alpha1.ConfigurationList, error) {
		client, err := DaprClient()
		if err != nil {
			return nil, err
		}

		list, err := client.ConfigurationV1alpha1().Configurations(namespace).List(meta_v1.ListOptions{})
		// This means that the Dapr Configurations CRD is not installed and
		// therefore no configuration items exist.
		if apierrors.IsNotFound(err) {
			list = &v1alpha1.ConfigurationList{
				Items: []v1alpha1.Configuration{},
			}
		} else if err != nil {
			return nil, err
		}

		return list, err
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

		if name == "" || strings.EqualFold(confName, name) {
			filtered = append(filtered, c)
			filteredSpecs = append(filteredSpecs, configurationDetailedOutput{
				Name:      confName,
				Namespace: c.GetNamespace(),
				Spec:      c.Spec,
			})
		}
	}

	if outputFormat == "" || outputFormat == "list" {
		return printConfigurationList(writer, filtered)
	}

	return utils.PrintDetail(writer, outputFormat, filteredSpecs)
}

func printConfigurationList(writer io.Writer, list []v1alpha1.Configuration) error {
	co := []configurationsOutput{}
	for _, c := range list {
		co = append(co, configurationsOutput{
			TracingEnabled: tracingEnabled(c.Spec.TracingSpec),
			Name:           c.GetName(),
			Namespace:      c.GetNamespace(),
			MetricsEnabled: c.Spec.MetricSpec.Enabled,
			Created:        c.CreationTimestamp.Format("2006-01-02 15:04.05"),
			Age:            age.GetAge(c.CreationTimestamp.Time),
		})
	}

	return utils.MarshalAndWriteTable(writer, co)
}

func tracingEnabled(spec v1alpha1.TracingSpec) bool {
	sr, err := strconv.ParseFloat(spec.SamplingRate, 32)
	if err != nil {
		return false
	}
	return sr > 0
}
