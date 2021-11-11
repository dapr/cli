// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	k8s "k8s.io/client-go/kubernetes"

	scheme "github.com/dapr/dapr/pkg/client/clientset/versioned"

	//  azure auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"

	//  gcp auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	//  oidc auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	//  openstack auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const kubeConfigDelimiter = ":"

var (
	doOnce     sync.Once
	kubeconfig *string
)

func init() {
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
}

func getConfig() (*rest.Config, error) {
	doOnce.Do(func() {
		flag.Parse()
	})
	kubeConfigEnv := os.Getenv("KUBECONFIG")
	delimiterBelongsToPath := strings.Count(*kubeconfig, kubeConfigDelimiter) == 1 && strings.EqualFold(*kubeconfig, kubeConfigEnv)

	if len(kubeConfigEnv) != 0 && !delimiterBelongsToPath {
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

// GetKubeConfigClient returns the kubeconfig and the client created from the kubeconfig.
func GetKubeConfigClient() (*rest.Config, *k8s.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, nil, err
	}
	client, err := k8s.NewForConfig(config)
	if err != nil {
		return config, nil, err
	}
	return config, client, nil
}

// Client returns a new Kubernetes client.
func Client() (*k8s.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	return k8s.NewForConfig(config)
}

// DaprClient returns a new Kubernetes Dapr client.
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
