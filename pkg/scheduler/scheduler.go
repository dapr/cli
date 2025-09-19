/*
Copyright 2025 The Dapr Authors
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

package scheduler

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/dapr/cli/pkg/kubernetes"
)

func etcdClient(kubernetesMode bool, schedulerNamespace string) (*clientv3.Client, context.CancelFunc, error) {
	var etcdClient *clientv3.Client
	var err error
	if kubernetesMode {
		var cancel context.CancelFunc
		etcdClient, cancel, err = etcdClientKubernetes(schedulerNamespace)
		if err != nil {
			return nil, nil, err
		}
		return etcdClient, cancel, nil
	} else {
		etcdClient, err = getEtcdClient("localhost:2379")
		if err != nil {
			return nil, nil, err
		}
	}

	return etcdClient, func() {}, nil
}

func getEtcdClient(host string) (*clientv3.Client, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{host},
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func etcdClientKubernetes(namespace string) (*clientv3.Client, context.CancelFunc, error) {
	config, _, err := kubernetes.GetKubeConfigClient()
	if err != nil {
		return nil, nil, err
	}

	portForward, err := kubernetes.NewPortForward(
		config,
		namespace,
		"dapr-scheduler-server-0",
		"localhost",
		2379,
		2379,
		false,
	)
	if err != nil {
		return nil, nil, err
	}

	if err = portForward.Init(); err != nil {
		return nil, nil, err
	}

	client, err := getEtcdClient("localhost:2379")
	if err != nil {
		return nil, nil, err
	}

	return client, portForward.Stop, nil
}
