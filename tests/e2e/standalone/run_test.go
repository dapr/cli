//go:build e2e
// +build e2e

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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandaloneRun(t *testing.T) {
	ensureDaprInstallation(t)

	for _, path := range getSocketCases() {
		t.Run(fmt.Sprintf("normal exit, socket: %s", path), func(t *testing.T) {
			output, err := cmdRun(path, "--", "bash", "-c", "echo test")
			t.Log(output)
			require.NoError(t, err, "run failed")
			assert.Contains(t, output, "Exited App successfully")
			assert.Contains(t, output, "Exited Dapr successfully")
		})

		t.Run(fmt.Sprintf("error exit, socket: %s", path), func(t *testing.T) {
			output, err := cmdRun(path, "--", "bash", "-c", "exit 1")
			t.Log(output)
			require.Error(t, err, "run failed")
			assert.Contains(t, output, "The App process exited with error code: exit status 1")
			assert.Contains(t, output, "Exited Dapr successfully")
		})

		t.Run("Use internal gRPC port if specified", func(t *testing.T) {
			output, err := cmdRun(path, "--dapr-internal-grpc-port", "9999", "--", "bash", "-c", "echo test")
			t.Log(output)
			require.NoError(t, err, "run failed")
			assert.Contains(t, output, "internal gRPC server is running on port 9999")
			assert.Contains(t, output, "Exited App successfully")
			assert.Contains(t, output, "Exited Dapr successfully")
		})
	}

	t.Run("API shutdown without socket", func(t *testing.T) {
		// Test that the CLI exits on a daprd shutdown.
		output, err := cmdRun("", "--dapr-http-port", "9999", "--", "bash", "-c", "curl -v -X POST http://localhost:9999/v1.0/shutdown; sleep 10; exit 1")
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully", "App should be shutdown before it has a chance to return non-zero")
		assert.Contains(t, output, "Exited Dapr successfully")
	})

	t.Run("API shutdown with socket", func(t *testing.T) {
		// Test that the CLI exits on a daprd shutdown.
		output, err := cmdRun("/tmp", "--app-id", "testapp", "--", "bash", "-c", "curl --unix-socket /tmp/dapr-testapp-http.socket -v -X POST http://unix/v1.0/shutdown; sleep 10; exit 1")
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited Dapr successfully")
	})
}
