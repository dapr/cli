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
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	k8s "k8s.io/client-go/kubernetes"

	"github.com/dapr/cli/pkg/age"
	"github.com/dapr/cli/pkg/print"
)

var controlPlaneLabels = []string{
	"dapr-operator",
	"dapr-sentry",
	"dapr-placement", // TODO: This is for the backward compatibility. Remove this after 1.0.0 GA release.
	"dapr-placement-server",
	"dapr-sidecar-injector",
	"dapr-dashboard",
	"dapr-scheduler-server",
}

type StatusClient struct {
	client k8s.Interface
}

// StatusOutput represents the status of a named Dapr resource.
type StatusOutput struct {
	Name      string `csv:"NAME"`
	Namespace string `csv:"NAMESPACE"`
	Healthy   string `csv:"HEALTHY"`
	Status    string `csv:"STATUS"`
	Replicas  int    `csv:"REPLICAS"`
	Version   string `csv:"VERSION"`
	Age       string `csv:"AGE"`
	Created   string `csv:"CREATED"`
}

// Create a new k8s client for status commands.
func NewStatusClient() (*StatusClient, error) {
	clientset, err := Client()
	if err != nil {
		return nil, err
	}
	return &StatusClient{
		client: clientset,
	}, nil
}

// List status for Dapr resources.
func (s *StatusClient) Status() ([]StatusOutput, error) {
	//nolint
	client := s.client
	if client == nil {
		return nil, errors.New("kubernetes client not initialized")
	}
	var wg sync.WaitGroup
	wg.Add(len(controlPlaneLabels))

	m := sync.Mutex{}
	statuses := []StatusOutput{}

	for _, lbl := range controlPlaneLabels {
		go func(label string) {
			defer wg.Done()
			// Query all namespaces for Dapr pods.
			p, err := ListPodsInterface(client, map[string]string{
				"app": label,
			})
			if err != nil {
				print.WarningStatusEvent(os.Stdout, "Failed to get status for %s: %s", label, err.Error())
				return
			}

			if len(p.Items) == 0 {
				return
			}
			pod := p.Items[0]
			replicas := len(p.Items)
			image := pod.Spec.Containers[0].Image
			namespace := pod.GetNamespace()
			age := age.GetAge(pod.CreationTimestamp.Time)
			created := pod.CreationTimestamp.Format("2006-01-02 15:04.05")

			// Version is part of the docker image tag which is expected to be present at the end of image uri.
			// expected format: <image>:<tag>. For example: daprio/dapr:1.8.0.
			// tag can be either <version> or <version>-<image-variant>. For example: 1.8.0-mariner.
			version := image[strings.LastIndex(image, ":")+1:]
			status := ""

			// loop through all replicas and update to Running/Healthy status only if all instances are Running and Healthy.
			healthy := "False"
			running := true

			for _, p := range p.Items {
				if len(p.Status.ContainerStatuses) == 0 {
					status = string(p.Status.Phase)
				} else if p.Status.ContainerStatuses[0].State.Waiting != nil {
					status = fmt.Sprintf("Waiting (%s)", p.Status.ContainerStatuses[0].State.Waiting.Reason)
				} else if pod.Status.ContainerStatuses[0].State.Terminated != nil {
					status = "Terminated"
				}

				if len(p.Status.ContainerStatuses) == 0 ||
					p.Status.ContainerStatuses[0].State.Running == nil {
					running = false

					break
				}

				if p.Status.ContainerStatuses[0].Ready {
					healthy = "True"
				}
			}

			if running {
				status = "Running"
			}

			s := StatusOutput{
				Name:      label,
				Namespace: namespace,
				Created:   created,
				Age:       age,
				Status:    status,
				Version:   version,
				Healthy:   healthy,
				Replicas:  replicas,
			}

			m.Lock()
			statuses = append(statuses, s)
			m.Unlock()
		}(lbl)
	}

	wg.Wait()
	return statuses, nil
}
