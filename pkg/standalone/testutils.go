// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
)

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
