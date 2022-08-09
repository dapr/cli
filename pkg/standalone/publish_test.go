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
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dapr/cli/utils"
)

func TestPublish(t *testing.T) {
	testCases := []struct {
		name          string
		publishAppID  string
		pubsubName    string
		payload       []byte
		topic         string
		lo            ListOutput
		listErr       error
		postResponse  string
		handler       http.HandlerFunc
		errorExpected bool
		errString     string
	}{
		{
			name:          "test empty appID",
			publishAppID:  "",
			payload:       []byte("test"),
			pubsubName:    "test",
			errString:     "publishAppID is missing",
			errorExpected: true,
			handler:       handlerTestPathResp("", ""),
		},
		{
			name:          "test empty topic",
			publishAppID:  "test",
			payload:       []byte("test"),
			pubsubName:    "test",
			errString:     "topic is missing",
			errorExpected: true,
			handler:       handlerTestPathResp("", ""),
		},
		{
			name:          "test empty pubsubName",
			publishAppID:  "test",
			payload:       []byte("test"),
			topic:         "test",
			errString:     "pubsubName is missing",
			errorExpected: true,
			handler:       handlerTestPathResp("", ""),
		},
		{
			name:          "test list error",
			publishAppID:  "test",
			payload:       []byte("test"),
			topic:         "test",
			pubsubName:    "test",
			listErr:       assert.AnError,
			errString:     assert.AnError.Error(),
			errorExpected: true,
			handler:       handlerTestPathResp("", ""),
		},
		{
			name:         "test empty appID in list output",
			publishAppID: "test",
			payload:      []byte("test"),
			topic:        "test",
			pubsubName:   "test",
			lo: ListOutput{
				// empty appID.
				Command: "test",
			},
			errString:     "couldn't find a running Dapr instance",
			errorExpected: true,
			handler:       handlerTestPathResp("", ""),
		},
		{
			name:         "successful call not found",
			publishAppID: "myAppID",
			pubsubName:   "testPubsubName",
			topic:        "testTopic",
			payload:      []byte("test payload"),
			lo: ListOutput{
				AppID: "not my myAppID",
			},
			errString:     "couldn't find a running Dapr instance",
			errorExpected: true,
			handler:       handlerTestPathResp("", ""),
		},
		{
			name:         "successful call",
			publishAppID: "myAppID",
			pubsubName:   "testPubsubName",
			topic:        "testTopic",
			payload:      []byte("test payload"),
			postResponse: "test payload",
			lo: ListOutput{
				AppID: "myAppID",
			},
			handler: handlerTestPathResp("/v1.0/publish/testPubsubName/testTopic", ""),
		},
		{
			name:         "successful cloudevent envelope",
			publishAppID: "myAppID",
			pubsubName:   "testPubsubName",
			topic:        "testTopic",
			payload:      []byte(`{"id": "1234", "source": "test", "specversion": "1.0", "type": "product.v1", "datacontenttype": "application/json", "data": {"id": "test", "description": "Testing 12345"}}`),
			postResponse: "test payload",
			lo: ListOutput{
				AppID: "myAppID",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Content-Type") != "application/cloudevents+json" {
					w.WriteHeader(http.StatusInternalServerError)

					return
				}
				if r.Method == http.MethodGet {
					w.Write([]byte(""))
				} else {
					buf := new(bytes.Buffer)
					buf.ReadFrom(r.Body)
					w.Write(buf.Bytes())
				}
			},
		},
	}
	for _, socket := range []string{"", "/tmp"} {
		// TODO(@daixiang0): add Windows support.
		if runtime.GOOS == "windows" && socket != "" {
			continue
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if socket != "" {
					ts, l := getTestSocketServerFunc(tc.handler, tc.publishAppID, socket)
					go ts.Serve(l)
					defer func() {
						l.Close()
						for _, protocol := range []string{"http", "grpc"} {
							os.Remove(utils.GetSocket(socket, tc.publishAppID, protocol))
						}
					}()
				} else {
					ts, port := getTestServerFunc(tc.handler)
					ts.Start()
					defer ts.Close()
					tc.lo.HTTPPort = port
				}

				client := &Standalone{
					process: &mockDaprProcess{
						Lo:  []ListOutput{tc.lo},
						Err: tc.listErr,
					},
				}
				err := client.Publish(tc.publishAppID, tc.pubsubName, tc.topic, tc.payload, socket, nil)
				if tc.errorExpected {
					assert.Error(t, err, "expected an error")
					assert.Equal(t, tc.errString, err.Error(), "expected error strings to match")
				} else {
					assert.NoError(t, err, "expected no error")
				}
			})
		}
	}
}

func TestGetQueryParams(t *testing.T) {
	testCases := []struct {
		metadata map[string]interface{}
	}{
		{
			metadata: nil,
		},
		{
			metadata: map[string]interface{}{},
		},
		{
			metadata: map[string]interface{}{
				"rawPayload": "true",
			},
		},
		{
			metadata: map[string]interface{}{
				"rawPayload":   "true",
				"ttlInSeconds": "10",
			},
		},
	}

	for _, tc := range testCases {
		queryParams := getQueryParams(tc.metadata)

		if queryParams != "" {
			assert.True(t, queryParams[0] == '?', "expected query params to start with '?'")
			queryParams = queryParams[1:]
		}

		// since map is unordered, test for each metadata entry in the query params.
		for k, v := range tc.metadata {
			entry := fmt.Sprintf("metadata.%v=%v", k, v)
			assert.True(t, strings.Contains(queryParams, entry), "expected query params to contain %s", entry)
			queryParams = strings.Replace(queryParams, entry, "", 1)
		}

		// finally, check that the query params don't contain any unexpected entries.
		// after removing the expected entries, the remaining string should only contain "&".
		assert.Equal(t, len(queryParams), strings.Count(queryParams, "&"), "expected query params to not contain any unexpected entries")
	}
}
