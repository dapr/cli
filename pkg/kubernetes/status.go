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

var controlPlaneLabels = []string{"dapr-operator", "dapr-sentry", "dapr-placement", "dapr-sidecar-injector"}

// StatusOutput represents the status of a named Dapr resource.
type StatusOutput struct {
	Name      string `csv:"NAME"`
	Namespace string `csv:"NAMESPACE"`
	Healthy   string `csv:"HEALTHY"`
	Status    string `csv:"STATUS"`
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
			if err == nil && len(p.Items) == 1 {
				pod := p.Items[0]

				image := pod.Spec.Containers[0].Image
				namespace := pod.GetNamespace()
				age := age.GetAge(pod.CreationTimestamp.Time)
				created := pod.CreationTimestamp.Format("2006-01-02 15:04.05")
				version := image[strings.IndexAny(image, ":")+1:]
				status := ""

				if pod.Status.ContainerStatuses[0].State.Waiting != nil {
					status = fmt.Sprintf("Waiting (%s)", pod.Status.ContainerStatuses[0].State.Waiting.Reason)
				} else if pod.Status.ContainerStatuses[0].State.Running != nil {
					status = "Running"
				} else if pod.Status.ContainerStatuses[0].State.Terminated != nil {
					status = "Terminated"
				}

				healthy := "False"
				if pod.Status.ContainerStatuses[0].Ready {
					healthy = "True"
				}

				s := StatusOutput{
					Name:      label,
					Namespace: namespace,
					Created:   created,
					Age:       age,
					Status:    status,
					Version:   version,
					Healthy:   healthy,
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
