package kubernetes

import (
	"context"
	"strings"

	core_v1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8s "k8s.io/client-go/kubernetes"
)

func ListPodsInterface(client k8s.Interface, labelSelector map[string]string) (*core_v1.PodList, error) {
	opts := v1.ListOptions{}
	if labelSelector != nil {
		opts.LabelSelector = labels.FormatLabels(labelSelector)
	}
	return client.CoreV1().Pods(v1.NamespaceAll).List(context.TODO(), opts)
}

func ListPods(client *k8s.Clientset, namespace string, labelSelector map[string]string) (*core_v1.PodList, error) {
	opts := v1.ListOptions{}
	if labelSelector != nil {
		opts.LabelSelector = labels.FormatLabels(labelSelector)
	}
	return client.CoreV1().Pods(namespace).List(context.TODO(), opts)
}

// CheckPodExists returns a boolean representing the pod's existence and the namespace that the given pod resides in,
// or empty if not present in the given namespace.
func CheckPodExists(client *k8s.Clientset, namespace string, labelSelector map[string]string, deployName string) (bool, string) {
	opts := v1.ListOptions{}
	if labelSelector != nil {
		opts.LabelSelector = labels.FormatLabels(labelSelector)
	}

	podList, err := client.CoreV1().Pods(namespace).List(context.TODO(), opts)
	if err != nil {
		return false, ""
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase == core_v1.PodRunning {
			if strings.HasPrefix(pod.Name, deployName) {
				return true, pod.Namespace
			}
		}
	}
	return false, ""
}
