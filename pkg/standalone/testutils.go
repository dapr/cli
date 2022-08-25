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

package standalone

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/dapr/cli/utils"
)

const SocketFormat = "/tmp/dapr-%s-http.socket"

type mockDaprProcess struct {
	Lo  []ListOutput
	Err error
}

func (m *mockDaprProcess) List() ([]ListOutput, error) {
	return m.Lo, m.Err
}

func getTestServerFunc(handler http.Handler) (*httptest.Server, int) {
	ts := httptest.NewUnstartedServer(handler)

	return ts, ts.Listener.Addr().(*net.TCPAddr).Port
}

func getTestServer(expectedPath, resp string) (*httptest.Server, int) {
	ts := httptest.NewUnstartedServer(handlerTestPathResp(expectedPath, resp))

	return ts, ts.Listener.Addr().(*net.TCPAddr).Port
}

func getTestSocketServerFunc(handler http.Handler, appID, path string) (*http.Server, net.Listener) {
	s := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: time.Duration(5) * time.Second,
	}

	socket := utils.GetSocket(path, appID, "http")
	l, err := net.Listen("unix", socket)
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen on %v: %v", socket, err))
	}
	return s, l
}

func getTestSocketServer(expectedPath, resp, appID, path string) (*http.Server, net.Listener) {
	s := &http.Server{
		Handler:           handlerTestPathResp(expectedPath, resp),
		ReadHeaderTimeout: time.Duration(5) * time.Second,
	}

	socket := utils.GetSocket(path, appID, "http")
	l, err := net.Listen("unix", socket)
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen on %v: %v", socket, err))
	}
	return s, l
}

func handlerTestPathResp(expectedPath, resp string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if expectedPath != "" && r.RequestURI != expectedPath {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
		if r.Method == http.MethodGet {
			w.Write([]byte(resp))
		} else {
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)
			w.Write(buf.Bytes())
		}
	}
}
