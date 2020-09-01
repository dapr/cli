// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dapr/cli/pkg/age"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	controlPlaneLabels = []string{"dapr-operator", "dapr-sentry", "dapr-placement", "dapr-sidecar-injector"}
)

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

// List status for Dapr resources.
func Status() ([]StatusOutput, error) {
	client, err := Client()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	wg.Add(len(controlPlaneLabels))

	m := sync.Mutex{}
	statuses := []StatusOutput{}

	for _, lbl := range controlPlaneLabels {
		go func(label string) {
			p, err := ListPods(client, v1.NamespaceAll, map[string]string{
				"app": label,
			})
			if err == nil {
				pod := p.Items[0]
				replicas := len(p.Items)
				image := pod.Spec.Containers[0].Image
				namespace := pod.GetNamespace()
				age := age.GetAge(pod.CreationTimestamp.Time)
				created := pod.CreationTimestamp.Format("2006-01-02 15:04.05")
				version := image[strings.IndexAny(image, ":")+1:]
				status := ""

				// loop through all replicas and update to Running/Healthy status only if all instances are Running and Healthy
				healthy := "False"
				running := true

				for _, p := range p.Items {
					if p.Status.ContainerStatuses[0].State.Waiting != nil {
						status = fmt.Sprintf("Waiting (%s)", p.Status.ContainerStatuses[0].State.Waiting.Reason)
					} else if pod.Status.ContainerStatuses[0].State.Terminated != nil {
						status = "Terminated"
					}

					if p.Status.ContainerStatuses[0].State.Running == nil {
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
			}
			wg.Done()
		}(lbl)
	}

	wg.Wait()
	return statuses, nil
}
