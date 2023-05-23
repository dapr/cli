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
	"google.golang.org/grpc/metadata"
)

func TestStandaloneInvoke(t *testing.T) {
	ensureDaprInstallation(t)
	s := daprHttp.NewService(":9987")

	err := s.AddServiceInvocationHandler("/test", func(ctx context.Context, e *common.InvocationEvent) (*common.Content, error) {
		val := &common.Content{
			Data:        e.Data,
			ContentType: e.ContentType,
			DataTypeURL: e.DataTypeURL,
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			values := md.Get("Some-Header")
			if len(values) > 0 {
				val.Data = []byte(values[0])
			}
		}
		return val, nil
	})

	assert.NoError(t, err, "unable to AddTopicEventHandler")

	defer s.Stop()
	go func() {
		err = s.Start()

		// ignore server closed errors.
		if err == http.ErrServerClosed {
			err = nil
		}

		assert.NoError(t, err, "unable to listen on :9987")
	}()

	for _, path := range getSocketCases() {
		executeAgainstRunningDapr(t, func() {
			t.Run(fmt.Sprintf("data from file with socket %s", path), func(t *testing.T) {
				output, err := cmdInvoke("invoke_e2e", "test", path, "--data-file", "../testdata/message.json")
				t.Log(output)
				assert.NoError(t, err, "unable to invoke with  --data-file")
				assert.Contains(t, output, "App invoked successfully")
				assert.Contains(t, output, "{\"dapr\": \"is_great\"}")
			})

			t.Run(fmt.Sprintf("data from string with socket %s", path), func(t *testing.T) {
				output, err := cmdInvoke("invoke_e2e", "test", path, "--data", "{\"cli\": \"is_working\"}")
				t.Log(output)
				assert.NoError(t, err, "unable to invoke with --data")
				assert.Contains(t, output, "{\"cli\": \"is_working\"}")
				assert.Contains(t, output, "App invoked successfully")
			})

			t.Run(fmt.Sprintf("data from non-existent file fails with socket %s", path), func(t *testing.T) {
				output, err := cmdInvoke("invoke_e2e", "test", path, "--data-file", "a/file/that/does/not/exist")
				t.Log(output)
				assert.Error(t, err, "a non-existent --data-file should fail with error")
				assert.Contains(t, output, "Error reading payload from 'a/file/that/does/not/exist'. Error: ")
			})

			t.Run(fmt.Sprintf("invoke only one of data and data-file with socket %s", path), func(t *testing.T) {
				output, err := cmdInvoke("invoke_e2e", "test", path, "--data-file", "../testdata/message.json", "--data", "{\"cli\": \"is_working\"}")
				t.Log(output)
				assert.Error(t, err, "--data and --data-file should not be allowed together")
				assert.Contains(t, output, "Only one of --data and --data-file allowed in the same invoke command")
			})

			t.Run(fmt.Sprintf("invoke an invalid app %s", path), func(t *testing.T) {
				output, err := cmdInvoke("invoke_e2e_2", "test", path)
				t.Log(output)
				assert.Error(t, err, "app invoke_e2e_2 should not exist")
				assert.Contains(t, output, "error invoking app invoke_e2e_2: app ID invoke_e2e_2 not found")
			})

			t.Run(fmt.Sprintf("invoke with an invalid method name %s", path), func(t *testing.T) {
				output, err := cmdInvoke("invoke_e2e", "test2", path)
				t.Log(output)
				assert.Error(t, err, "method test2 should not exist")
				assert.Contains(t, output, "error invoking app invoke_e2e: 404 Not Found")
			})

			t.Run(fmt.Sprintf("invoke mehod %s with http headers", path), func(t *testing.T) {
				output, err := cmdInvoke("invoke_e2e", "test", path, "--header", "Some-Header=aValue")
				t.Log(output)
				assert.NoError(t, err, "")
				assert.Contains(t, output, "aValue")
			})

			output, err := cmdStopWithAppID("invoke_e2e")
			t.Log(output)
			require.NoError(t, err, "dapr stop failed")
			assert.Contains(t, output, "app stopped successfully: invoke_e2e")
		}, "run", "--app-id", "invoke_e2e", "--app-port", "9987", "--unix-domain-socket", path)
	}
}
