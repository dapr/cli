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
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utiltesting "k8s.io/client-go/util/testing"
)

func newDaprAppPod(name string, namespace string, appName string, creationTime time.Time, appPort string, httpPort string, grpcPort string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{},
			Labels: map[string]string{
				"app": appName,
			},
			CreationTimestamp: metav1.Time{
				Time: creationTime,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{},
				{
					Name: "daprd",
					Args: []string{
						"--mode",
						"kubernetes",
						"--dapr-http-port",
						httpPort,
						"--dapr-grpc-port",
						grpcPort,
						"--dapr-internal-grpc-port",
						"50002",
						"--dapr-listen-addresses",
						"[::1],127.0.0.1",
						"--dapr-public-port",
						"3501",
						"--app-port",
						appPort,
						"--app-id",
						appName,
						"--control-plane-address",
						"dapr-api.keel-system.svc.cluster.local:80",
						"--app-protocol",
						"http",
						"--placement-host-address",
						"dapr-placement-server.keel-system.svc.cluster.local:50005",
						"--config",
						"testAppID-Config",
						"--log-level",
						"info",
						"--app-max-concurrency",
						"-1",
						"--sentry-address",
						"dapr-sentry.keel-system.svc.cluster.local:80",
						"--enable-metrics=true",
						"--metrics-port",
						"9090",
						"--dapr-http-max-request-size",
						"-1",
						"--enable-mtls",
					},
				},
			},
		},
	}
}

func Test_getAppInfo(t *testing.T) {
	client := fake.NewSimpleClientset(newDaprAppPod(
		"testAppPod", "testAppNameSpace",
		"testAppID", time.Now(),
		"8080", "80801", "80802"))

	testCases := []struct {
		name          string
		errorExpected bool
		errString     string
		appID         string
		want          *AppInfo
	}{
		{
			name:          "get test Pod",
			appID:         "testAppID",
			errorExpected: false,
			errString:     "",
			want: &AppInfo{
				AppID: "testAppID", HTTPPort: "80801", GRPCPort: "80802", AppPort: "8080", PodName: "testAppPod", Namespace: "testAppNameSpace",
			},
		},
		{
			name:          "get error Pod",
			appID:         "errorAppID",
			errorExpected: true,
			errString:     "errorAppID not found",
			want:          nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appInfo, err := GetAppInfo(client, tc.appID)
			if tc.errorExpected {
				assert.Error(t, err, "expected an error")
				assert.Equal(t, tc.errString, err.Error(), "expected error strings to match")
			} else {
				assert.NoError(t, err, "expected no error")
				assert.Equal(t, tc.want, appInfo, "expected appInfo to match")
			}
		})
	}
}

func Test_invoke(t *testing.T) {
	app := &AppInfo{
		AppID: "testAppID", AppPort: "8080", HTTPPort: "3500", GRPCPort: "50001", PodName: "testAppPod", Namespace: "testAppNameSpace",
	}

	testCases := []struct {
		name          string
		errorExpected bool
		errString     string
		appID         string
		method        string
		verb          string
		data          []byte
		URLExpected   string
	}{
		{
			name:          "get request",
			errorExpected: false,
			errString:     "",
			method:        "hello",
			verb:          "GET",
			data:          nil,
			URLExpected: "https://localhost/api/v1/" +
				"namespaces/testAppNameSpace/pods/testAppPod:8080/proxy/" +
				"hello",
		},
		{
			name:          "get request",
			errorExpected: false,
			errString:     "",
			method:        "hello?abc=123&cdr=345#abb=aaa",
			verb:          "GET",
			data:          nil,
			URLExpected: "https://localhost/api/v1/" +
				"namespaces/testAppNameSpace/pods/testAppPod:8080/proxy/" +
				"hello?abc=123&cdr=345#abb=aaa",
		},
		{
			name:          "post request",
			errorExpected: false,
			errString:     "",
			method:        "hello?abc=123&cdr=345#abb=aaa",
			verb:          "POST",
			data:          []byte("hello"),
			URLExpected: "https://localhost/api/v1/" +
				"namespaces/testAppNameSpace/pods/testAppPod:8080/proxy/" +
				"hello?abc=123&cdr=345#abb=aaa",
		},
		{
			name:          "post request",
			errorExpected: false,
			errString:     "errorAppID not found",
			method:        "hello",
			verb:          "POST",
			data:          []byte("hello"),
			URLExpected: "https://localhost/api/v1/" +
				"namespaces/testAppNameSpace/pods/testAppPod:8080/proxy/" +
				"hello",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testServer, fakeHandler := testServerEnv(t, 200)
			defer testServer.Close()
			client, err := restClient(testServer)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			_, err = invoke(client, app, tc.method, tc.data, tc.verb)
			if tc.errorExpected {
				assert.Error(t, err, "expected an error")
				assert.Equal(t, tc.errString, err.Error(), "expected error strings to match")
			} else {
				assert.NoError(t, err, "expected no error")
				data := string(tc.data)
				fakeHandler.ValidateRequest(t, tc.URLExpected, tc.verb, &data)
			}
		})
	}
}

func testServerEnv(t *testing.T, statusCode int) (*httptest.Server, *utiltesting.FakeHandler) {
	t.Helper()
	fakeHandler := utiltesting.FakeHandler{
		StatusCode:   statusCode,
		ResponseBody: "",
		T:            t,
	}
	testServer := httptest.NewServer(&fakeHandler)
	return testServer, &fakeHandler
}

func restClient(testServer *httptest.Server) (*rest.RESTClient, error) {
	c, err := rest.RESTClientFor(&rest.Config{
		Host: testServer.URL,
		ContentConfig: rest.ContentConfig{
			GroupVersion:         &v1.SchemeGroupVersion,
			NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		},
		APIPath:  "api",
		Username: "user",
		Password: "pass",
	})
	return c, err
}
