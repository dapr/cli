// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dapr/cli/utils"
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

	for _, socket := range []string{"", "/tmp"} {
		// TODO(@daixiang0): add Windows support
		if runtime.GOOS == "windows" && socket != "" {
			continue
		}
		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%s get, socket: %v", tc.name, socket), func(t *testing.T) {
				if socket != "" {
					ts, l := getTestSocketServer(tc.expectedPath, tc.resp, tc.appID, socket)
					go ts.Serve(l)
					defer func() {
						l.Close()
						for _, protocol := range []string{"http", "grpc"} {
							os.Remove(utils.GetSocket(socket, tc.appID, protocol))
						}
					}()
				} else {
					ts, port := getTestServer(tc.expectedPath, tc.resp)
					ts.Start()
					defer ts.Close()
					tc.lo.HTTPPort = port
				}

				client := &Standalone{
					process: &mockDaprProcess{
						Lo: []ListOutput{
							tc.lo,
						},
						Err: tc.listErr,
					},
				}

				res, err := client.Invoke(tc.appID, tc.method, []byte(tc.resp), "GET", socket)
				if tc.errorExpected {
					assert.Error(t, err, "expected an error")
					assert.Equal(t, tc.errString, err.Error(), "expected error strings to match")
				} else {
					assert.NoError(t, err, "expected no error")
					assert.Equal(t, tc.resp, res, "expected response to match")
				}
			})

			t.Run(fmt.Sprintf("%s post, socket: %v", tc.name, socket), func(t *testing.T) {
				if socket != "" {
					ts, l := getTestSocketServer(tc.expectedPath, tc.resp, tc.appID, socket)
					go ts.Serve(l)
					defer func() {
						l.Close()
						for _, protocol := range []string{"http", "grpc"} {
							os.Remove(utils.GetSocket(socket, tc.appID, protocol))
						}
					}()
				} else {
					ts, port := getTestServer(tc.expectedPath, tc.resp)
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
				res, err := client.Invoke(tc.appID, tc.method, []byte(tc.resp), "POST", socket)
				if tc.errorExpected {
					assert.Error(t, err, "expected an error")
					assert.Equal(t, tc.errString, err.Error(), "expected error strings to match")
				} else {
					assert.NoError(t, err, "expected no error")
					assert.Equal(t, tc.resp, res, "expected response to match")
				}
			})

			t.Run(fmt.Sprintf("%s delete, socket: %v", tc.name, socket), func(t *testing.T) {
				if socket != "" {
					ts, l := getTestSocketServer(tc.expectedPath, tc.resp, tc.appID, socket)
					go ts.Serve(l)
					defer func() {
						l.Close()
						for _, protocol := range []string{"http", "grpc"} {
							os.Remove(utils.GetSocket(socket, tc.appID, protocol))
						}
					}()
				} else {
					ts, port := getTestServer(tc.expectedPath, tc.resp)
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
				res, err := client.Invoke(tc.appID, tc.method, []byte(tc.resp), "DELETE", socket)
				if tc.errorExpected {
					assert.Error(t, err, "expected an error")
					assert.Equal(t, tc.errString, err.Error(), "expected error strings to match")
				} else {
					assert.NoError(t, err, "expected no error")
					assert.Equal(t, tc.resp, res, "expected response to match")
				}
			})

			t.Run(fmt.Sprintf("%s put, socket: %v", tc.name, socket), func(t *testing.T) {
				if socket != "" {
					ts, l := getTestSocketServer(tc.expectedPath, tc.resp, tc.appID, socket)
					go ts.Serve(l)
					defer func() {
						l.Close()
						for _, protocol := range []string{"http", "grpc"} {
							os.Remove(utils.GetSocket(socket, tc.appID, protocol))
						}
					}()
				} else {
					ts, port := getTestServer(tc.expectedPath, tc.resp)
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
				res, err := client.Invoke(tc.appID, tc.method, []byte(tc.resp), "PUT", socket)
				if tc.errorExpected {
					assert.Error(t, err, "expected an error")
					assert.Equal(t, tc.errString, err.Error(), "expected error strings to match")
				} else {
					assert.NoError(t, err, "expected no error")
					assert.Equal(t, tc.resp, res, "expected response to match")
				}
			})
		}
	}
}
