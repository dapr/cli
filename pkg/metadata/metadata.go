// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package metadata

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dapr/cli/pkg/api"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

// Get retrieves the metadata of a given app's sidecar.
func Get(httpPort int) (*api.Metadata, error) {
	url := makeMetadataGetEndpoint(httpPort)
	// nolint:gosec
	r, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	defer r.Body.Close()
	return handleMetadataResponse(r)
}

// Put sets one metadata attribute on a given app's sidecar.
func Put(httpPort int, key, value string) error {
	client := retryablehttp.NewClient()
	client.Logger = nil
	url := makeMetadataPutEndpoint(httpPort, key)

	req, err := retryablehttp.NewRequest("PUT", url, strings.NewReader(value))
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	r, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	defer r.Body.Close()
	return nil
}

func makeMetadataGetEndpoint(httpPort int) string {
	return fmt.Sprintf("http://127.0.0.1:%v/v%s/metadata", httpPort, api.RuntimeAPIVersion)
}

func makeMetadataPutEndpoint(httpPort int, key string) string {
	return fmt.Sprintf("http://127.0.0.1:%v/v%s/metadata/%s", httpPort, api.RuntimeAPIVersion, key)
}

func handleMetadataResponse(response *http.Response) (*api.Metadata, error) {
	rb, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	var m api.Metadata
	err = json.Unmarshal(rb, &m)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}
	return &m, nil
}
