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
	"context"
	"fmt"
	"io"
	"os"

	corev1 "k8s.io/api/core/v1"
)

const (
	daprdContainerName    = "daprd"
	appIDContainerArgName = "--app-id"
)

// Logs fetches Dapr sidecar logs from Kubernetes.
func Logs(appID, podName, namespace string) error {
	client, err := Client()
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = corev1.NamespaceDefault
	}

	pods, err := ListPods(client, namespace, nil)
	if err != nil {
		return fmt.Errorf("could not get logs %w", err)
	}

	if podName == "" {
		// no pod name specified. in case of multiple pods, the first one will be selected.
		var foundDaprPod bool
		for _, pod := range pods.Items {
			if foundDaprPod {
				break
			}
			for _, container := range pod.Spec.Containers {
				if container.Name == daprdContainerName {
					// find app ID.
					for i, arg := range container.Args {
						if arg == appIDContainerArgName {
							id := container.Args[i+1]
							if id == appID {
								podName = pod.Name
								foundDaprPod = true
								break
							}
						}
					}
				}
			}
		}
		if !foundDaprPod {
			return fmt.Errorf("could not get logs. Please check app-id (%s) and namespace (%s)", appID, namespace)
		}
	}

	getLogsRequest := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{Container: daprdContainerName, Follow: false})
	logStream, err := getLogsRequest.Stream(context.TODO())
	if err != nil {
		return fmt.Errorf("could not get logs. Please check pod-name (%s). Error - %w", podName, err)
	}
	defer logStream.Close()
	_, err = io.Copy(os.Stdout, logStream)
	if err != nil {
		return fmt.Errorf("could not get logs %w", err)
	}

	return nil
}
