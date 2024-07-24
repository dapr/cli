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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/dapr/cli/pkg/api"
	"github.com/dapr/cli/utils"
)

// Publish publishes payload to topic in pubsub referenced by pubsubName.
func (s *Standalone) Publish(publishAppID, pubsubName, topic string, payload []byte, socket string, metadata map[string]interface{}) error {
	if publishAppID == "" {
		return errors.New("publishAppID is missing")
	}

	if pubsubName == "" {
		return errors.New("pubsubName is missing")
	}

	if topic == "" {
		return errors.New("topic is missing")
	}

	queryParams := getQueryParams(metadata)

	l, err := s.process.List()
	if err != nil {
		return err
	}

	instance, err := getDaprInstance(l, publishAppID)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://unix/v%s/publish/%s/%s%s", api.RuntimeAPIVersion, pubsubName, topic, queryParams)

	var httpc http.Client
	if socket != "" {
		httpc.Transport = &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", utils.GetSocket(socket, publishAppID, "http"))
			},
		}
	} else {
		url = fmt.Sprintf("http://localhost:%d/v%s/publish/%s/%s%s", instance.HTTPPort, api.RuntimeAPIVersion, pubsubName, topic, queryParams)
	}

	contentType := "application/json"

	// Detect publishing with CloudEvents envelope.
	var cloudEvent map[string]interface{}
	if json.Unmarshal(payload, &cloudEvent); err == nil {
		_, hasID := cloudEvent["id"]
		_, hasSource := cloudEvent["source"]
		_, hasSpecVersion := cloudEvent["specversion"]
		_, hasType := cloudEvent["type"]
		_, hasData := cloudEvent["data"]
		if hasID && hasSource && hasSpecVersion && hasType && hasData {
			contentType = "application/cloudevents+json"
		}
	}

	r, err := httpc.Post(url, contentType, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode >= 300 || r.StatusCode < 200 {
		fmt.Println(url)
		return fmt.Errorf("unexpected status code %d on publishing to %s in %s", r.StatusCode, topic, pubsubName)
	}

	return nil
}

func getDaprInstance(list []ListOutput, publishAppID string) (ListOutput, error) {
	for i := 0; i < len(list); i++ {
		if list[i].AppID == publishAppID {
			return list[i], nil
		}
	}
	return ListOutput{}, errors.New("couldn't find a running Dapr instance")
}

// getQueryParams returns the HTTP query parameter from the metadata map.
// It appends the prefix "metadata." to each key.
// The return value includes the "?" prefix if metadata is not empty.
func getQueryParams(metadata map[string]interface{}) string {
	queryParams := ""
	for k, v := range metadata {
		queryParams += fmt.Sprintf("metadata.%v=%v&", k, v)
	}
	// Prefix with "?" and remove the last "&".
	if queryParams != "" {
		queryParams = fmt.Sprintf("?%s", queryParams[:len(queryParams)-1])
	}
	return queryParams
}
