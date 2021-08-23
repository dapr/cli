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
	"github.com/dapr/cli/pkg/age"
)

// ListOutput represents the application ID, application port and creation time.
type ListOutput struct {
	Namespace string `csv:"NAMESPACE" json:"namespace" yaml:"namespace"`
	AppID     string `csv:"APP ID"    json:"appId"     yaml:"appId"`
	AppPort   string `csv:"APP PORT"  json:"appPort"   yaml:"appPort"`
	Age       string `csv:"AGE"       json:"age"       yaml:"age"`
	Created   string `csv:"CREATED"   json:"created"   yaml:"created"`
}

// List outputs all the applications.
func List(namespace string) ([]ListOutput, error) {
	client, err := Client()
	if err != nil {
		return nil, err
	}

	podList, err := ListPods(client, namespace, nil)
	if err != nil {
		return nil, err
	}

	l := []ListOutput{}
	for _, p := range podList.Items {
		for _, c := range p.Spec.Containers {
			if c.Name == "daprd" {
				lo := ListOutput{}
				for i, a := range c.Args {
					if a == "--app-port" {
						port := c.Args[i+1]
						lo.AppPort = port
					} else if a == "--app-id" {
						id := c.Args[i+1]
						lo.AppID = id
					}
				}
				lo.Namespace = p.GetNamespace()
				lo.Created = p.CreationTimestamp.Format("2006-01-02 15:04.05")
				lo.Age = age.GetAge(p.CreationTimestamp.Time)
				l = append(l, lo)
			}
		}
	}

	return l, nil
}
