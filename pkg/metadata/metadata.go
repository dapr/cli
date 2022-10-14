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

package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	retryablehttp "github.com/hashicorp/go-retryablehttp"

	"github.com/dapr/cli/pkg/api"
	"github.com/dapr/cli/utils"
)

// Get retrieves the metadata of a given app's sidecar.
func Get(httpPort int, appID, socket string) (*api.Metadata, error) {
	url := makeMetadataGetEndpoint(httpPort)

	var httpc http.Client
	if socket != "" {
		fileInfo, err := os.Stat(socket)
		if err != nil {
			return nil, err
		}

		if fileInfo.IsDir() {
			socket = utils.GetSocket(socket, appID, "http")
		}

		httpc.Transport = &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socket)
			},
		}
	}

	r, err := httpc.Get(url)
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()
	return handleMetadataResponse(r)
}

// Put sets one metadata attribute on a given app's sidecar.
func Put(httpPort int, key, value, appID, socket string) error {
	client := retryablehttp.NewClient()
	client.Logger = nil

	if socket != "" {
		client.HTTPClient.Transport = &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", utils.GetSocket(socket, appID, "http"))
			},
		}
	}

	url := makeMetadataPutEndpoint(httpPort, key)

	req, err := retryablehttp.NewRequest(http.MethodPut, url, strings.NewReader(value))
	if err != nil {
		return err
	}

	r, err := client.Do(req)
	if err != nil {
		return err
	}

	defer r.Body.Close()
	if socket != "" {
		// Retryablehttp does not close idle socket connections.
		defer client.HTTPClient.CloseIdleConnections()
	}
	return nil
}

func makeMetadataGetEndpoint(httpPort int) string {
	if httpPort == 0 {
		return fmt.Sprintf("http://unix/v%s/metadata", api.RuntimeAPIVersion)
	}
	return fmt.Sprintf("http://127.0.0.1:%v/v%s/metadata", httpPort, api.RuntimeAPIVersion)
}

func makeMetadataPutEndpoint(httpPort int, key string) string {
	if httpPort == 0 {
		return fmt.Sprintf("http://unix/v%s/metadata/%s", api.RuntimeAPIVersion, key)
	}
	return fmt.Sprintf("http://127.0.0.1:%v/v%s/metadata/%s", httpPort, api.RuntimeAPIVersion, key)
}

func handleMetadataResponse(response *http.Response) (*api.Metadata, error) {
	rb, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var m api.Metadata
	err = json.Unmarshal(rb, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
