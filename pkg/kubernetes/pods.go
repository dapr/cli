package kubernetes

import (
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
