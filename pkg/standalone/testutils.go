// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"

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
		Handler: handler,
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
		Handler: handlerTestPathResp(expectedPath, resp),
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
