// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/dapr/cli/pkg/api"
	"github.com/dapr/cli/utils"
)

// Invoke is a command to invoke a remote or local dapr instance.
func (s *Standalone) Invoke(appID, method string, data []byte, verb string, path string) (string, error) {
	list, err := s.process.List()
	if err != nil {
		return "", err
	}

	for _, lo := range list {
		if lo.AppID == appID {
			url := makeEndpoint(lo, method)
			req, err := http.NewRequest(verb, url, bytes.NewBuffer(data))
			if err != nil {
				return "", err
			}
			req.Header.Set("Content-Type", "application/json")

			var httpc http.Client

			if path != "" {
				httpc.Transport = &http.Transport{
					DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
						return net.Dial("unix", utils.GetSocket(path, appID, "http"))
					},
				}
			}

			r, err := httpc.Do(req)
			if err != nil {
				return "", err
			}
			defer r.Body.Close()
			return handleResponse(r)
		}
	}

	return "", fmt.Errorf("app ID %s not found", appID)
}

func makeEndpoint(lo ListOutput, method string) string {
	return fmt.Sprintf("http://127.0.0.1:%s/v%s/invoke/%s/method/%s", fmt.Sprintf("%v", lo.HTTPPort), api.RuntimeAPIVersion, lo.AppID, method)
}

func handleResponse(response *http.Response) (string, error) {
	rb, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if len(rb) > 0 {
		return string(rb), nil
	}

	return "", nil
}
