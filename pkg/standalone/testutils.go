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

func getTestServer(expectedPath, resp string) (*httptest.Server, int) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(
		w http.ResponseWriter, r *http.Request) {
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
	}))

	return ts, ts.Listener.Addr().(*net.TCPAddr).Port
}

func getTestSocketServer(expectedPath, resp, appID, path string) (*http.Server, net.Listener) {
	s := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		}),
	}

	socket := utils.GetSocket(path, appID, "http")
	l, err := net.Listen("unix", socket)
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen on %v: %v", socket, err))
	}
	return s, l
}
