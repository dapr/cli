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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	kubeConfigDelimiter := ":"
	if runtime.GOOS == "windows" {
		kubeConfigDelimiter = ";"
	}
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
		return nil, fmt.Errorf("error: %w", err)
	}
	return config, nil
}

// GetKubeConfigClient returns the kubeconfig and the client created from the kubeconfig.
func GetKubeConfigClient() (*rest.Config, *k8s.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("error: %w", err)
	}
	client, err := k8s.NewForConfig(config)
	if err != nil {
		return config, nil, fmt.Errorf("error: %w", err)
	}
	return config, client, nil
}

// Client returns a new Kubernetes client.
func Client() (*k8s.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}
	//nolint
	return k8s.NewForConfig(config)
}

// DaprClient returns a new Kubernetes Dapr client.
func DaprClient() (scheme.Interface, error) {
	config, err := getConfig()
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}
	//nolint
	return scheme.NewForConfig(config)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
