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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/dapr/cli/pkg/print"
)

const (
	daprdContainerName    = "daprd"
	appIDContainerArgName = "--app-id"

	maxListingRetry = 10
	listingDelay    = 200 * time.Microsecond
	streamingDelay  = 100 * time.Millisecond
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

// streamContainerLogsToDisk streams all containers logs for the given selector to a given disk directory.
func streamContainerLogsToDisk(ctx context.Context, appID string, appLogWriter, daprdLogWriter io.Writer, podClient v1.PodInterface) error {
	var err error
	var podList *corev1.PodList
	counter := 0
	for {
		podList, err = getPods(ctx, appID, podClient)
		if err != nil {
			return fmt.Errorf("error listing the pod with label %s=%s: %w", daprAppIDKey, appID, err)
		}
		if len(podList.Items) != 0 {
			break
		}
		counter++
		if counter == maxListingRetry {
			return fmt.Errorf("error getting logs: error listing the pod with label %s=%s after %d retires", daprAppIDKey, appID, maxListingRetry)
		}
		// Retry after a delay.
		time.Sleep(listingDelay)
	}

	for _, pod := range podList.Items {
		print.InfoStatusEvent(os.Stdout, "Streaming logs for containers in pod %q", pod.GetName())
		for _, container := range pod.Spec.Containers {
			fileWriter := daprdLogWriter
			if container.Name != daprdContainerName {
				fileWriter = appLogWriter
			}

			// create a go routine for each container to stream logs into file/console.
			go func(pod, containerName, appID string, fileWriter io.Writer) {
			loop:
				for {
					req := podClient.GetLogs(pod, &corev1.PodLogOptions{
						Container: containerName,
						Follow:    true,
					})
					stream, err := req.Stream(ctx)
					if err != nil {
						switch {
						case strings.Contains(err.Error(), "Pending"):
							// Retry after a delay.
							time.Sleep(streamingDelay)
							continue loop
						case strings.Contains(err.Error(), "ContainerCreating"):
							// Retry after a delay.
							time.Sleep(streamingDelay)
							continue loop
						case errors.Is(err, context.Canceled):
							return
						default:
							return
						}
					}
					defer stream.Close()

					if containerName != daprdContainerName {
						streamScanner := bufio.NewScanner(stream)
						for streamScanner.Scan() {
							fmt.Fprintln(fileWriter, print.Blue(fmt.Sprintf("== APP - %s == %s", appID, streamScanner.Text())))
						}
					} else {
						_, err = io.Copy(fileWriter, stream)
						if err != nil {
							switch {
							case errors.Is(err, context.Canceled):
								return
							default:
								return
							}
						}
					}

					return
				}
			}(pod.GetName(), container.Name, appID, fileWriter)
		}
	}

	return nil
}

func getPods(ctx context.Context, appID string, podClient v1.PodInterface) (*corev1.PodList, error) {
	listCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	labelSelector := fmt.Sprintf("%s=%s", daprAppIDKey, appID)
	fmt.Println("Select", labelSelector)
	podList, err := podClient.List(listCtx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	cancel()
	if err != nil {
		return nil, err
	}
	return podList, nil
}
