package kubernetes

import (
	"strings"

	core_v1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8s "k8s.io/client-go/kubernetes"
)

func ListPods(client *k8s.Clientset, namespace string, labelSelector map[string]string) (*core_v1.PodList, error) {
	opts := v1.ListOptions{}
	if labelSelector != nil {
		opts.LabelSelector = labels.FormatLabels(labelSelector)
	}
	return client.CoreV1().Pods(v1.NamespaceAll).List(opts)
}

// PodLocation returns the namespace that the given pod resides in, or empty if not in the given possibilities list
func PodLocation(client *k8s.Clientset, labelSelector map[string]string, deployName string, namespacesToSearch []string) string {
	opts := v1.ListOptions{}
	if labelSelector != nil {
		opts.LabelSelector = labels.FormatLabels(labelSelector)
	}

	for _, nspace := range namespacesToSearch {
		podList, err := client.CoreV1().Pods(nspace).List(opts)
		if err != nil {
			return nspace
		}

		for _, pod := range podList.Items {
			if pod.Status.Phase == core_v1.PodRunning {
				if strings.HasPrefix(pod.Name, deployName) {
					return nspace
				}
			}
		}
	}
	return ""
}
