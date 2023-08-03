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
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	k8s "k8s.io/client-go/kubernetes"
)

const podWatchErrTemplate = "error creating pod watcher"

var errPodUnknown error = errors.New("pod in unknown/failed state")

func ListPodsInterface(client k8s.Interface, labelSelector map[string]string) (*corev1.PodList, error) {
	opts := metav1.ListOptions{}
	if labelSelector != nil {
		opts.LabelSelector = labels.FormatLabels(labelSelector)
	}
	return client.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), opts)
}

func ListPods(client *k8s.Clientset, namespace string, labelSelector map[string]string) (*corev1.PodList, error) {
	opts := metav1.ListOptions{}
	if labelSelector != nil {
		opts.LabelSelector = labels.FormatLabels(labelSelector)
	}
	return client.CoreV1().Pods(namespace).List(context.TODO(), opts)
}

// CheckPodExists returns a boolean representing the pod's existence and the namespace that the given pod resides in,
// or empty if not present in the given namespace.
func CheckPodExists(client k8s.Interface, namespace string, labelSelector map[string]string, deployName string) (bool, string) {
	opts := metav1.ListOptions{}
	if labelSelector != nil {
		opts.LabelSelector = labels.FormatLabels(labelSelector)
	}

	podList, err := client.CoreV1().Pods(namespace).List(context.TODO(), opts)
	if err != nil {
		return false, ""
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			if strings.HasPrefix(pod.Name, deployName) {
				return true, pod.Namespace
			}
		}
	}
	return false, ""
}

func createPodWatcher(ctx context.Context, client k8s.Interface, namespace, appID string) (watch.Interface, error) {
	labelSelector := fmt.Sprintf("%s=%s", daprAppIDKey, appID)

	opts := metav1.ListOptions{
		TypeMeta:      metav1.TypeMeta{},
		LabelSelector: labelSelector,
	}

	return client.CoreV1().Pods(namespace).Watch(ctx, opts)
}

func waitPodDeleted(ctx context.Context, client k8s.Interface, namespace, appID string) error {
	watcher, err := createPodWatcher(ctx, client, namespace, appID)
	if err != nil {
		return fmt.Errorf("%s : %w", podWatchErrTemplate, err)
	}

	defer watcher.Stop()

	for {
		select {
		case event := <-watcher.ResultChan():

			if event.Type == watch.Deleted {
				return nil
			}

		case <-ctx.Done():
			return fmt.Errorf("error context cancelled while waiting for pod deletion: %w", context.Canceled)
		}
	}
}

func waitPodRunning(ctx context.Context, client k8s.Interface, namespace, appID string) error {
	watcher, err := createPodWatcher(ctx, client, namespace, appID)
	if err != nil {
		return fmt.Errorf("%s : %w", podWatchErrTemplate, err)
	}

	defer watcher.Stop()

	for {
		select {
		case event := <-watcher.ResultChan():
			pod := event.Object.(*corev1.Pod)

			if pod.Status.Phase == corev1.PodRunning {
				return nil
			} else if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodUnknown {
				return fmt.Errorf("error waiting for pod run: %w", errPodUnknown)
			}

		case <-ctx.Done():
			return fmt.Errorf("error context cancelled while waiting for pod run: %w", context.Canceled)
		}
	}
}
