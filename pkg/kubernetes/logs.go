// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"fmt"
	"io"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const daprdContainerName = "daprd"
const daprIDContainerArgName = "--dapr-id"
const logsForDaprOption = "dapr"
const logsForAppOption = "app"

// Logs fetches Dapr app logs from Kubernetes.
func Logs(appID, _for, namespace string) error {

	client, err := Client()
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = corev1.NamespaceDefault
	}

	pods, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Could not get logs %v", err)
	}
	var podName string
	var foundDaprPod bool
	for _, pod := range pods.Items {
		if foundDaprPod {
			break
		}
		for _, container := range pod.Spec.Containers {
			if container.Name == daprdContainerName {
				//find app ID
				for i, arg := range container.Args {
					if arg == daprIDContainerArgName {
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
		return fmt.Errorf("Could not get logs. Please check app-id (%s) and namespace (%s)", appID, namespace)
	}
	var containerName string

	if _for == logsForDaprOption {
		containerName = daprdContainerName
	} else if _for == logsForAppOption { //app logs are needed
		pod, err := client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Could not get logs %v", err)
		}
		for _, container := range pod.Spec.Containers {
			if container.Name != daprdContainerName {
				containerName = container.Name
				break
			}
		}
	}
	if containerName == "" {
		return fmt.Errorf("Could not get logs. Please check the command")
	}

	//fmt.Printf("Getting logs for container %s in pod %s\n", containerName, podName)

	getLogsRequest := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{Container: containerName, Follow: false})
	logStream, err := getLogsRequest.Stream()
	if err != nil {
		return fmt.Errorf("Could not get logs %v", err)
	}
	defer logStream.Close()
	_, err = io.Copy(os.Stdout, logStream)
	if err != nil {
		return fmt.Errorf("Could not get logs %v", err)
	}

	return nil
}
