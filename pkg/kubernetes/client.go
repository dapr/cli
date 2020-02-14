// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	scheme "github.com/dapr/dapr/pkg/client/clientset/versioned"
	k8s "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Add auth
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const kubeConfigDelimiter = ":"

func getConfig() (*rest.Config, error) {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	kubeConfigEnv := os.Getenv("KUBECONFIG")
	if len(kubeConfigEnv) != 0 {
		kubeConfigs := strings.Split(kubeConfigEnv, kubeConfigDelimiter)
		if len(kubeConfigs) > 1 {
			return nil, fmt.Errorf("multiple kubeconfigs in KUBECONFIG environment variable - %s", kubeConfigEnv)
		}
		kubeconfig = &kubeConfigs[0]
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// Client returns a new Kubernetes client.
func Client() (*k8s.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	return k8s.NewForConfig(config)
}

// DaprClient returns a new Kubernetes Dapr client
func DaprClient() (scheme.Interface, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	return scheme.NewForConfig(config)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
