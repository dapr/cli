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
	"context"
	"fmt"
	"github.com/dapr/cli/pkg/api"
	"net/url"
	"strings"

	core_v1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/net"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type AppInfo struct {
	AppID     string `csv:"APP ID"      json:"appId"        yaml:"appId"`
	HTTPPort  string `csv:"HTTP PORT"   json:"httpPort"     yaml:"httpPort"`
	GRPCPort  string `csv:"GRPC PORT"   json:"grpcPort"     yaml:"grpcPort"`
	AppPort   string `csv:"APP PORT"    json:"appPort"      yaml:"appPort"`
	PodName   string `csv:"POD NAME"    json:"podName"      yaml:"podName"`
	Namespace string `csv:"NAMESPACE"   json:"namespace"    yaml:"namespace"`
}

type (
	DaprPod     core_v1.Pod
	DaprAppList []*AppInfo
)

// Invoke is a command to invoke a remote or local dapr instance.
func Invoke(appID, method string, data []byte, verb string) (string, error) {
	client, err := Client()
	if err != nil {
		return "", err
	}

	app, err := GetAppInfo(client, appID)
	if err != nil {
		return "", err
	}

	return invoke(client.CoreV1().RESTClient(), app, method, data, verb)
}

func invoke(client rest.Interface, app *AppInfo, method string, data []byte, verb string) (string, error) {
	res, err := app.Request(client.Verb(verb), method, data, verb)
	if err != nil {
		return "", fmt.Errorf("error get request: %w", err)
	}

	result := res.Do(context.TODO())
	rawbody, err := result.Raw()
	if err != nil {
		return "", fmt.Errorf("error get raw: %w", err)
	}

	if len(rawbody) > 0 {
		return string(rawbody), nil
	}

	return "", nil
}

func GetAppInfo(client k8s.Interface, appID string) (*AppInfo, error) {
	list, err := ListAppInfos(client, appID)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("%s not found", appID)
	}
	app := list[0]
	return app, nil
}

// List outputs plugins.
func ListAppInfos(client k8s.Interface, appIDs ...string) (DaprAppList, error) {
	opts := v1.ListOptions{}
	podList, err := client.CoreV1().Pods(v1.NamespaceAll).List(context.TODO(), opts)
	if err != nil {
		return nil, fmt.Errorf("err get pods list:%w", err)
	}

	fn := func(*AppInfo) bool {
		return true
	}
	if len(appIDs) > 0 {
		fn = func(a *AppInfo) bool {
			for _, id := range appIDs {
				if id != "" && a.AppID == id {
					return true
				}
			}
			return false
		}
	}

	l := make(DaprAppList, 0)
	for _, p := range podList.Items {
		p := DaprPod(p)
	FindLoop:
		for _, c := range p.Spec.Containers {
			if c.Name == "daprd" {
				app := getAppInfoFromPod(&p)
				if fn(app) {
					l = append(l, app)
				}
				break FindLoop
			}
		}
	}

	return l, nil
}

func getAppInfoFromPod(p *DaprPod) (a *AppInfo) {
	for _, c := range p.Spec.Containers {
		if c.Name == "daprd" {
			a = &AppInfo{
				PodName:   p.Name,
				Namespace: p.Namespace,
			}
			for i, arg := range c.Args {
				if arg == "--app-port" {
					port := c.Args[i+1]
					a.AppPort = port
				} else if arg == "--dapr-http-port" {
					port := c.Args[i+1]
					a.HTTPPort = port
				} else if arg == "--dapr-grpc-port" {
					port := c.Args[i+1]
					a.GRPCPort = port
				} else if arg == "--app-id" {
					id := c.Args[i+1]
					a.AppID = id
				}
			}
		}
	}

	return
}

func (a *AppInfo) Request(r *rest.Request, method string, data []byte, verb string) (*rest.Request, error) {
	r = r.Namespace(a.Namespace).
		Resource("pods").
		SubResource("proxy").
		SetHeader("Content-Type", "application/json").
		Name(net.JoinSchemeNamePort("", a.PodName, a.HTTPPort))
	if data != nil {
		r = r.Body(data)
	}

	u, err := url.Parse(method)
	if err != nil {
		return nil, fmt.Errorf("error parse method %s: %w", method, err)
	}

	suffix := fmt.Sprintf("v%s/invoke/%s/method/%s", api.RuntimeAPIVersion, a.AppID, u.Path)
	r = r.Suffix(suffix)

	for k, vs := range u.Query() {
		r = r.Param(k, strings.Join(vs, ","))
	}

	return r, nil
}
