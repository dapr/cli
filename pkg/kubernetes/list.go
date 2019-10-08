// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"github.com/dapr/cli/pkg/age"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ListOutput struct {
	AppID   string `csv:"APP ID"`
	AppPort string `csv:"APP PORT"`
	Age     string `csv:"AGE"`
	Created string `csv:"CREATED"`
}

func List() ([]ListOutput, error) {
	client, err := Client()
	if err != nil {
		return nil, err
	}

	podList, err := client.CoreV1().Pods(core_v1.NamespaceAll).List(meta_v1.ListOptions{})
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
					} else if a == "--dapr-id" {
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
