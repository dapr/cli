// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublish(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name          string
		publishAppID  string
		pubsubName    string
		payload       []byte
		topic         string
		lo            ListOutput
		listErr       error
		expectedPath  string
		postResponse  string
		resp          string
		errorExpected bool
		errString     string
	}{
		{
			name:          "test empty topic",
			publishAppID:  "",
			payload:       []byte("test"),
			pubsubName:    "test",
			errString:     "publishAppID is missing",
			errorExpected: true,
		},
		{
			name:          "test empty topic",
			publishAppID:  "test",
			payload:       []byte("test"),
			pubsubName:    "test",
			errString:     "topic is missing",
			errorExpected: true,
		},
		{
			name:          "test empty pubsubName",
			publishAppID:  "test",
			payload:       []byte("test"),
			topic:         "test",
			errString:     "pubsubName is missing",
			errorExpected: true,
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
		},
		{
			name:         "test empty appID in list output",
			publishAppID: "test",
			payload:      []byte("test"),
			topic:        "test",
			pubsubName:   "test",
			lo: ListOutput{
				// empty appID
				Command: "test",
			},
			errString:     "couldn't find a running Dapr instance",
			errorExpected: true,
		},
		{
			name:         "successful call",
			publishAppID: "myAppID",
			pubsubName:   "testPubsubName",
			topic:        "testTopic",
			payload:      []byte("test payload"),
			lo: ListOutput{
				AppID: "not my myAppID",
			},
			errString:     "couldn't find a running Dapr instance",
			errorExpected: true,
		},
		{
			name:         "successful call",
			publishAppID: "myAppID",
			pubsubName:   "testPubsubName",
			topic:        "testTopic",
			payload:      []byte("test payload"),
			expectedPath: "/v1.0/publish/testPubsubName/testTopic",
			postResponse: "test payload",
			lo: ListOutput{
				AppID: "myAppID",
			},
		},
	}
	//nolint
	for _, tc := range testCases {
		t.Parallel()
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ts, port := getTestServer(tc.expectedPath, tc.resp)
			ts.Start()
			defer ts.Close()
			tc.lo.HTTPPort = port
			client := &Standalone{
				process: &mockDaprProcess{
					Lo:  []ListOutput{tc.lo},
					Err: tc.listErr,
				},
			}
			err := client.Publish(tc.publishAppID, tc.pubsubName, tc.topic, tc.payload)
			if tc.errorExpected {
				assert.Error(t, err, "expected an error")
				assert.Equal(t, tc.errString, err.Error(), "expected error strings to match")
			} else {
				assert.NoError(t, err, "expected no error")
			}
		})
	}
}
