// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package invoke

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dapr/cli/pkg/api"
	"github.com/dapr/cli/pkg/standalone"
)

// Get invokes the application via HTTP GET.
func Get(appID, method string) (string, error) {
	list, err := standalone.List()
	if err != nil {
		return "", err
	}
	for _, lo := range list {
		if lo.AppID == appID {
			url := makeEndpoint(lo, method)
			// nolint:gosec
			r, err := http.Get(url)
			if err != nil {
				return "", err
			}

			defer r.Body.Close()
			return handleResponse(r)
		}
	}

	return "", fmt.Errorf("app ID %s not found", appID)
}

// Post invokes the application via HTTP POST.
func Post(appID, method, payload string) (string, error) {
	list, err := standalone.List()
	if err != nil {
		return "", err
	}

	for _, lo := range list {
		if lo.AppID == appID {
			url := makeEndpoint(lo, method)
			// nolint: gosec
			r, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(payload)))
			if err != nil {
				return "", err
			}

			defer r.Body.Close()
			return handleResponse(r)
		}
	}

	return "", fmt.Errorf("app ID %s not found", appID)
}

func makeEndpoint(lo standalone.ListOutput, method string) string {
	return fmt.Sprintf("http://localhost:%s/v%s/invoke/%s/method/%s", fmt.Sprintf("%v", lo.HTTPPort), api.RuntimeAPIVersion, lo.AppID, method)
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
