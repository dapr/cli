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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	core_v1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// PortForward provides a port-forward connection in a kubernetes cluster.
type PortForward struct {
	Config     *rest.Config
	Method     string
	URL        *url.URL
	Host       string
	LocalPort  int
	RemotePort int
	EmitLogs   bool
	StopCh     chan struct{}
	ReadyCh    chan struct{}
}

// NewPortForward returns an instance of PortForward struct that can be used
// for establishing port-forwarding connection to a pod in kubernetes cluster,
// specified by namespace and deployName.
func NewPortForward(
	config *rest.Config,
	namespace, deployName string,
	host string, localPort, remotePort int,
	emitLogs bool,
) (*PortForward, error) {
	client, err := k8s.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("can't create Clientset for %q: %w", deployName, err)
	}

	podList, err := ListPods(client, namespace, nil)
	if err != nil {
		return nil, err
	}

	podName := ""

	for _, pod := range podList.Items {
		if pod.Status.Phase == core_v1.PodRunning {
			if strings.HasPrefix(pod.Name, deployName) {
				podName = pod.Name
				break
			}
		}
	}

	if podName == "" {
		return nil, fmt.Errorf("no running pods found for %s", deployName)
	}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	return &PortForward{
		Config:     config,
		Method:     "POST",
		URL:        req.URL(),
		Host:       host,
		LocalPort:  localPort,
		RemotePort: remotePort,
		EmitLogs:   emitLogs,
		StopCh:     make(chan struct{}, 1),
		ReadyCh:    make(chan struct{}),
	}, nil
}

// Init creates and runs a port-forward connection.
// This function blocks until connection is established.
// Note: Caller should call Stop() to finish the connection.
func (pf *PortForward) Init() error {
	transport, upgrader, err := spdy.RoundTripperFor(pf.Config)
	if err != nil {
		return fmt.Errorf("cannot connect to Kubernetes cluster: %w", err)
	}

	out := io.Discard
	errOut := io.Discard
	if pf.EmitLogs {
		out = os.Stdout
		errOut = os.Stderr
	}

	ports := []string{fmt.Sprintf("%d:%d", pf.LocalPort, pf.RemotePort)}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, pf.Method, pf.URL)

	fw, err := portforward.NewOnAddresses(dialer, []string{pf.Host}, ports, pf.StopCh, pf.ReadyCh, out, errOut)
	if err != nil {
		return fmt.Errorf("cannot create PortForwarder: %w", err)
	}

	failure := make(chan error)
	go func() {
		if err := fw.ForwardPorts(); err != nil {
			failure <- err
		}
	}()

	select {
	// if `fw.ForwardPorts()` succeeds, block until terminated.
	case <-pf.ReadyCh:
		ports, err := fw.GetPorts()
		if err != nil {
			return fmt.Errorf("can not get the local and remote ports: %w", err)
		}
		if len(ports) == 0 {
			return fmt.Errorf("can not get the local and remote ports: error getting ports length")
		}

		pf.LocalPort = int(ports[0].Local)
		pf.RemotePort = int(ports[0].Remote)

	// if failure, causing a receive `<-failure` and returns the error.
	case err := <-failure:
		return err
	}

	return nil
}

// Stop terminates port-forwarding connection.
func (pf *PortForward) Stop() {
	close(pf.StopCh)
}

// GetStop returns StopCh for a PortForward instance.
// Receiving on StopCh will block until the port forwarding stops.
func (pf *PortForward) GetStop() <-chan struct{} {
	return pf.StopCh
}
