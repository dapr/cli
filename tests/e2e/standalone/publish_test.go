//go:build e2e && !template
// +build e2e,!template

/*
Copyright 2022 The Dapr Authors
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

package standalone_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/dapr/go-sdk/service/common"
	daprHttp "github.com/dapr/go-sdk/service/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandalonePublish(t *testing.T) {
	ensureDaprInstallation(t)
	sub := &common.Subscription{
		PubsubName: "pubsub",
		Topic:      "sample",
		Route:      "/orders",
	}

	rawSub := &common.Subscription{
		PubsubName: "pubsub",
		Topic:      "raw-sample",
		Route:      "/raw-orders",
		Metadata: map[string]string{
			"rawPayload": "true",
		},
	}

	s := daprHttp.NewService(":9988")

	events := make(chan *common.TopicEvent)

	err := s.AddTopicEventHandler(sub, func(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
		events <- e
		return false, nil
	})

	err = s.AddTopicEventHandler(rawSub, func(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
		events <- e
		return false, nil
	})

	assert.NoError(t, err, "unable to AddTopicEventHandler")

	defer s.Stop()
	go func() {
		err = s.Start()

		// ignore server closed errors.
		if err == http.ErrServerClosed {
			err = nil
		}

		assert.NoError(t, err, "unable to listen on :9988")
	}()

	for _, path := range getSocketCases() {
		executeAgainstRunningDapr(t, func() {
			t.Run(fmt.Sprintf("publish message from file with socket %s", path), func(t *testing.T) {
				output, err := cmdPublish("pub_e2e", "pubsub", "sample", path, "--data-file", "../testdata/message.json")
				t.Log(output)
				assert.NoError(t, err, "unable to publish from --data-file")
				assert.Contains(t, output, "Event published successfully")

				event := <-events
				assert.Equal(t, map[string]interface{}{"dapr": "is_great"}, event.Data)
			})

			t.Run(fmt.Sprintf("publish cloudevent from file with socket %s", path), func(t *testing.T) {
				output, err := cmdPublish("pub_e2e", "pubsub", "sample", path, "--data-file", "../testdata/cloudevent.json")
				t.Log(output)
				assert.NoError(t, err, "unable to publish from --data-file")
				assert.Contains(t, output, "Event published successfully")

				event := <-events
				assert.Equal(t, &common.TopicEvent{
					ID:              "3cc97064-edd1-49f4-b911-c959a7370e68",
					Source:          "e2e_test",
					SpecVersion:     "1.0",
					Type:            "test.v1",
					DataContentType: "application/json",
					Subject:         "e2e_subject",
					PubsubName:      "pubsub",
					Topic:           "sample",
					Data:            map[string]interface{}{"dapr": "is_great"},
					RawData:         []byte(`{"dapr":"is_great"}`),
				}, event)
			})

			t.Run(fmt.Sprintf("publish from string with socket %s", path), func(t *testing.T) {
				output, err := cmdPublish("pub_e2e", "pubsub", "sample", path, "--data", "{\"cli\": \"is_working\"}")
				t.Log(output)
				assert.NoError(t, err, "unable to publish from --data")
				assert.Contains(t, output, "Event published successfully")

				event := <-events
				assert.Equal(t, map[string]interface{}{"cli": "is_working"}, event.Data)
			})

			t.Run(fmt.Sprintf("publish from non-existent file fails with socket %s", path), func(t *testing.T) {
				output, err := cmdPublish("pub_e2e", "pubsub", "sample", path, "--data-file", "a/file/that/does/not/exist")
				t.Log(output)
				assert.Contains(t, output, "Error reading payload from 'a/file/that/does/not/exist'. Error: ")
				assert.Error(t, err, "a non-existent --data-file should fail with error")
			})

			t.Run(fmt.Sprintf("publish only one of data and data-file with socket %s", path), func(t *testing.T) {
				output, err := cmdPublish("pub_e2e", "pubsub", "sample", path, "--data-file", "../testdata/message.json", "--data", "{\"cli\": \"is_working\"}")
				t.Log(output)
				assert.Error(t, err, "--data and --data-file should not be allowed together")
				assert.Contains(t, output, "Only one of --data and --data-file allowed in the same publish command")
			})

			t.Run("publish with invalid metadata fails", func(t *testing.T) {
				output, err := cmdPublish("pub_e2e", "pubsub", "raw-sample", path, "--data", "{\"cli\": \"is_working\"}", "--metadata", "not a valid JSON")
				t.Log(output)
				assert.Error(t, err, "invalid metadata should fail")
				assert.Contains(t, output, "Error parsing metadata as JSON")
			})

			t.Run("publish message without cloud event using metadata with socket", func(t *testing.T) {
				output, err := cmdPublish("pub_e2e", "pubsub", "raw-sample", path, "--data", "{\"cli\": \"is_working\"}", "--metadata", "{\"rawPayload\": \"true\"}")
				t.Log(output)
				assert.NoError(t, err, "unable to publish with rawPayload --metadata")
				assert.Contains(t, output, "Event published successfully")

				event := <-events
				assert.Equal(t, []byte("{\"cli\": \"is_working\"}"), event.Data)
			})

			output, err := cmdStopWithAppID("pub_e2e")
			t.Log(output)
			require.NoError(t, err, "dapr stop failed")
			assert.Contains(t, output, "app stopped successfully: pub_e2e")
		}, "run", "--app-id", "pub_e2e", "--app-port", "9988", "--unix-domain-socket", path)
	}
}
