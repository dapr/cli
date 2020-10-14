// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/dapr/cli/pkg/api"
)

// Invoke is a command to invoke a remote or local dapr instance.
func (s *Standalone) Invoke(appID, method, payload, verb string) (string, error) {
	list, err := s.process.List()
	if err != nil {
		return "", err
	}

	for _, lo := range list {
		if lo.AppID == appID {
			url := makeEndpoint(lo, method)
			var body io.Reader

			if payload != "" {
				body = bytes.NewBuffer([]byte(payload))
			}
			req, err := http.NewRequest(verb, url, body)
			if err != nil {
				return "", err
			}
			req.Header.Set("Content-Type", "application/json")

			r, err := http.DefaultClient.Do(req)
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
