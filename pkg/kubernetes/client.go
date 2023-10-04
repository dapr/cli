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
	"sync"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	scheme "github.com/dapr/dapr/pkg/client/clientset/versioned"

	//  azure auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"

	//  gcp auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	//  oidc auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

var (
	doOnce     sync.Once
	kubeconfig *string
)

func init() {
	kubeconfig = flag.String(clientcmd.RecommendedConfigPathFlag, "", "absolute path to the kubeconfig file")
}

func getConfig() (*rest.Config, error) {
	doOnce.Do(func() {
		flag.Parse()
	})

	if *kubeconfig != "" {
		// Load `kubeconfig` from command line clientcmd.RecommendedConfigPathFlag(e.g. kubeconfig).
		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, err
		}
		return config, err
	}

	// Load `kubeconfig` from clientcmd.RecommendedConfigPathEnvVar(e.g. KUBECONFIG) or clientcmd.RecommendedHomeFile (e.g. %HOME/.kube/config).
	configLoadRules := clientcmd.NewDefaultClientConfigLoadingRules()
	startingConfig, err := configLoadRules.GetStartingConfig()
	if err != nil {
		return nil, err
	}
	config, err := clientcmd.NewDefaultClientConfig(*startingConfig, nil).ClientConfig()
	return config, err
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
