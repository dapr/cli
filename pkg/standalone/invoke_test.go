// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvoke(t *testing.T) {
	testCases := []struct {
		name          string
		errorExpected bool
		errString     string
		appID         string
		method        string
		lo            ListOutput
		listErr       error
		expectedPath  string
		postResponse  string
		resp          string
	}{
		{
			name:          "list apps error",
			errorExpected: true,
			errString:     assert.AnError.Error(),
			listErr:       assert.AnError,
		},
		{
			name:          "appID not found",
			errorExpected: true,
			appID:         "invalid",
			errString:     "app ID invalid not found",
			lo: ListOutput{
				AppID: "testapp",
			},
		},
		{
			name:   "appID found successful invoke empty response",
			appID:  "testapp",
			method: "test",
			lo: ListOutput{
				AppID: "testapp",
			},
		},
		{
			name:   "appID found successful invoke",
			appID:  "testapp",
			method: "test",
			lo: ListOutput{
				AppID: "testapp",
			},
			expectedPath: "/v1.0/invoke/testapp/method/test",
			postResponse: "test payload",
			resp:         "successful invoke",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name+" get", func(t *testing.T) {
			ts, port := getTestServer(tc.expectedPath, tc.resp)
			ts.Start()
			defer ts.Close()
			tc.lo.HTTPPort = port
			client := &Standalone{
				process: &mockDaprProcess{
					Lo: []ListOutput{
						tc.lo,
					},
					Err: tc.listErr,
				},
			}
			res, err := client.InvokeGet(tc.appID, tc.method)
			if tc.errorExpected {
				assert.Error(t, err, "expected an error")
				assert.Equal(t, tc.errString, err.Error(), "expected error strings to match")
			} else {
				assert.NoError(t, err, "expected no error")
				assert.Equal(t, tc.resp, res, "expected response to match")
			}
		})
		t.Run(tc.name+" post", func(t *testing.T) {
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
			res, err := client.InvokePost(tc.appID, tc.method, "test payload")
			if tc.errorExpected {
				assert.Error(t, err, "expected an error")
				assert.Equal(t, tc.errString, err.Error(), "expected error strings to match")
			} else {
				assert.NoError(t, err, "expected no error")
				assert.Equal(t, tc.postResponse, res, "expected response to match")
			}
		})
	}
}
