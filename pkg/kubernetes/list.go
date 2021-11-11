// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dapr/cli/pkg/age"
)

// ListOutput represents the application ID, application port and creation time.
type ListOutput struct {
	AppID   string `csv:"APP ID"   json:"appId"   yaml:"appId"`
	AppPort string `csv:"APP PORT" json:"appPort" yaml:"appPort"`
	Age     string `csv:"AGE"      json:"age"     yaml:"age"`
	Created string `csv:"CREATED"  json:"created" yaml:"created"`
}

// List outputs all the applications.
func List() ([]ListOutput, error) {
	client, err := Client()
	if err != nil {
		return nil, err
	}

	podList, err := ListPods(client, meta_v1.NamespaceAll, nil)
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
				lo.Created = p.CreationTimestamp.Format("2006-01-02 15:04.05")
				lo.Age = age.GetAge(p.CreationTimestamp.Time)
				l = append(l, lo)
			}
		}
	}

	return l, nil
}
