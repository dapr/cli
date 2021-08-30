// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

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
//nolint
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
		return fmt.Errorf("could not get logs %v", err)
	}

	if podName == "" {
		// no pod name specified. in case of multiple pods, the first one will be selected
		var foundDaprPod bool
		for _, pod := range pods.Items {
			if foundDaprPod {
				break
			}
			for _, container := range pod.Spec.Containers {
				if container.Name == daprdContainerName {
					// find app ID
					//nolint
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
		return fmt.Errorf("could not get logs. Please check pod-name (%s). Error - %v", podName, err)
	}
	defer logStream.Close()
	_, err = io.Copy(os.Stdout, logStream)
	if err != nil {
		return fmt.Errorf("could not get logs %v", err)
	}

	return nil
}
