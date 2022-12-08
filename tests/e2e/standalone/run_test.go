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
	"runtime"
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
		if runtime.GOOS == "windows" {
			t.Skip("Skipping API shutdown with socket test in Windows")
		}

		// Test that the CLI exits on a daprd shutdown.
		output, err := cmdRun("/tmp", "--app-id", "testapp", "--", "bash", "-c", "curl --unix-socket /tmp/dapr-testapp-http.socket -v -X POST http://unix/v1.0/shutdown; sleep 10; exit 1")
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited Dapr successfully")
	})

	t.Run(fmt.Sprintf("check enableAPILogging flag in enabled mode"), func(t *testing.T) {
		args := []string{
			"--app-id", "enableApiLogging_info",
			"--enable-api-logging",
			"--log-level", "info",
			"--", "bash", "-c", "echo 'test'",
		}

		output, err := cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "level=info msg=\"HTTP API Called: PUT /v1.0/metadata/appCommand")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
	})

	t.Run(fmt.Sprintf("check enableAPILogging flag in disabled mode"), func(t *testing.T) {
		args := []string{
			"--app-id", "enableApiLogging_info",
			"--", "bash", "-c", "echo 'test'",
		}

		output, err := cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
		assert.NotContains(t, output, "level=info msg=\"HTTP API Called: PUT /v1.0/metadata/appCommand\"")
	})

	t.Run(fmt.Sprintf("check run with log JSON enabled"), func(t *testing.T) {
		args := []string{
			"--app-id", "logjson",
			"--log-as-json",
			"--", "bash", "-c", "echo 'test'",
		}
		output, err := cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "{\"app_id\":\"logjson\"")
		assert.Contains(t, output, "\"type\":\"log\"")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
	})

	t.Run(fmt.Sprintf("check run with resources-path flag"), func(t *testing.T) {
		args := []string{
			"--app-id", "testapp",
			"--resources-path", "../testdata/nonexistentdir",
			"--", "bash", "-c", "echo 'test'",
		}
		output, err := cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "failed to load components: open ../testdata/nonexistentdir:")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")

		args = []string{
			"--app-id", "testapp",
			"--resources-path", "../testdata/resources",
			"--", "bash", "-c", "echo 'test'",
		}
		output, err = cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "component loaded. name: test-statestore, type: state.in-memory/v1")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
	})

	t.Run("run with unknown flags", func(t *testing.T) {
		output, err := cmdRun("", "--flag")
		require.Error(t, err, "expected error on run unknown flag")
		require.Contains(t, output, "Error: unknown flag: --flag\nUsage:", "expected usage to be printed")
		require.Contains(t, output, "-a, --app-id string", "expected usage to be printed")
		require.Contains(t, output, "The id for your application, used for service discovery", "expected usage to be printed")
	})
}
