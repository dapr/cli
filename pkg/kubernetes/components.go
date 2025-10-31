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
	"sort"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dapr/cli/pkg/age"
	"github.com/dapr/cli/utils"
	v1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/dapr/dapr/pkg/client/clientset/versioned"
)

// ComponentsOutput represent a Dapr component.
type ComponentsOutput struct {
	Namespace string `csv:"Namespace"`
	Name      string `csv:"Name"`
	Type      string `csv:"Type"`
	Version   string `csv:"VERSION"`
	Scopes    string `csv:"SCOPES"`
	Created   string `csv:"CREATED"`
	Age       string `csv:"AGE"`
}

// PrintComponents prints all Dapr components.
func PrintComponents(name, namespace, outputFormat string) error {
	return writeComponents(os.Stdout, func() (*v1alpha1.ComponentList, error) {
		client, err := DaprClient()
		if err != nil {
			return nil, err
		}

		return ListComponents(client, namespace)
	}, name, outputFormat)
}

func ListComponents(client versioned.Interface, namespace string) (*v1alpha1.ComponentList, error) {
	list, err := client.ComponentsV1alpha1().Components(namespace).List(meta_v1.ListOptions{})
	// This means that the Dapr Components CRD is not installed and
	// therefore no component items exist.
	if apierrors.IsNotFound(err) {
		list = &v1alpha1.ComponentList{
			Items: []v1alpha1.Component{},
		}
	} else if err != nil {
		return nil, err
	}

	return list, nil
}

func getComponent(client versioned.Interface, namespace string, componentName string) (*v1alpha1.Component, error) {
	c, err := client.ComponentsV1alpha1().Components(namespace).Get(componentName, meta_v1.GetOptions{})
	// This means that the Dapr Components CRD is not installed and
	// therefore no component items exist.
	if apierrors.IsNotFound(err) {
		return &v1alpha1.Component{}, nil
	} else if err != nil {
		return nil, err
	}
	return c, err
}

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
				Name:      confName,
				Namespace: c.GetNamespace(),
				Spec:      c.Spec,
			})
		}
	}

	if outputFormat == "" || outputFormat == "list" {
		return printComponentList(writer, filtered)
	}

	// filteredSpecs sort by namespace.
	sort.Slice(filteredSpecs, func(i, j int) bool {
		return filteredSpecs[i].Namespace > filteredSpecs[j].Namespace
	})
	return utils.PrintDetail(writer, outputFormat, filteredSpecs)
}

func printComponentList(writer io.Writer, list []v1alpha1.Component) error {
	co := []ComponentsOutput{}
	for _, c := range list {
		co = append(co, ComponentsOutput{
			Name:      c.GetName(),
			Namespace: c.GetNamespace(),
			Type:      c.Spec.Type,
			Created:   c.CreationTimestamp.Format("2006-01-02 15:04.05"),
			Age:       age.GetAge(c.CreationTimestamp.Time),
			Version:   c.Spec.Version,
			Scopes:    strings.Join(c.Scopes, ","),
		})
	}

	// co sort by namespace.
	sort.Slice(co, func(i, j int) bool {
		return co[i].Namespace > co[j].Namespace
	})
	return utils.MarshalAndWriteTable(writer, co)
}
