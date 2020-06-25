// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	core_v1 "k8s.io/api/core/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForward struct {
	Config     *rest.Config
	Method     string
	Url        *url.URL
	Host       string
	LocalPort  int
	RemotePort int
	EmitLogs   bool
	StopCh     chan struct{}
	ReadyCh    chan struct{}
}

func NewPortForward(
	config *rest.Config,
	namespace, deployName string,
	host string, localPort, remotePort int,
	emitLogs bool,
) (*PortForward, error) {
	time.Sleep(10 * time.Second)

	client, err := k8s.NewForConfig(config)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("No running pods found for %s", deployName)
	}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	return &PortForward{
		Config:     config,
		Method:     "POST",
		Url:        req.URL(),
		Host:       host,
		LocalPort:  localPort,
		RemotePort: remotePort,
		EmitLogs:   emitLogs,
		StopCh:     make(chan struct{}, 1),
		ReadyCh:    make(chan struct{}),
	}, nil

}

func (pf *PortForward) run() error {
	transport, upgrader, err := spdy.RoundTripperFor(pf.Config)
	if err != nil {
		return err
	}

	out := ioutil.Discard
	errOut := ioutil.Discard
	if pf.EmitLogs {
		out = os.Stdout
		errOut = os.Stderr
	}

	ports := []string{fmt.Sprintf("%d:%d", pf.LocalPort, pf.RemotePort)}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, pf.Method, pf.Url)

	fw, err := portforward.NewOnAddresses(dialer, []string{pf.Host}, ports, pf.StopCh, pf.ReadyCh, out, errOut)
	if err != nil {
		return err
	}

	return fw.ForwardPorts()
}

func (pf *PortForward) Init() error {
	failure := make(chan error)

	go func() {
		if err := pf.run(); err != nil {
			failure <- err
		}
	}()

	select {
	case <-pf.ReadyCh:
		// do nothing if port forwarding is initialized
	case err := <-failure:
		return err
	}

	return nil
}

func (pf *PortForward) Stop() {
	close(pf.StopCh)
}

func (pf *PortForward) GetStop() <-chan struct{} {
	return pf.StopCh
}
