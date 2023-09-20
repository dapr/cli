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
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandaloneRun(t *testing.T) {
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		// remove dapr installation after all tests in this function.
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})
	for _, path := range getSocketCases() {
		t.Run(fmt.Sprintf("normal exit, socket: %s", path), func(t *testing.T) {
			output, err := cmdRun(path, "--", "bash", "-c", "echo test")
			t.Log(output)
			require.NoError(t, err, "run failed")
			assert.Contains(t, output, "Exited App successfully")
			assert.Contains(t, output, "Exited Dapr successfully")
			assert.NotContains(t, output, "Could not update sidecar metadata for cliPID")
		})

		t.Run(fmt.Sprintf("error exit, socket: %s", path), func(t *testing.T) {
			output, err := cmdRun(path, "--", "bash", "-c", "exit 1")
			t.Log(output)
			require.Error(t, err, "run failed")
			assert.Contains(t, output, "The App process exited with error code: exit status 1")
			assert.Contains(t, output, "Exited Dapr successfully")
			assert.NotContains(t, output, "Could not update sidecar metadata for cliPID")
		})

		t.Run("Use internal gRPC port if specified", func(t *testing.T) {
			output, err := cmdRun(path, "--dapr-internal-grpc-port", "9999", "--", "bash", "-c", "echo test")
			t.Log(output)
			require.NoError(t, err, "run failed")
			assert.Contains(t, output, "Internal gRPC server is running on port 9999")
			assert.Contains(t, output, "Exited App successfully")
			assert.Contains(t, output, "Exited Dapr successfully")
			assert.NotContains(t, output, "Could not update sidecar metadata for cliPID")
		})
	}

	t.Run("API shutdown without socket", func(t *testing.T) {
		// Test that the CLI exits on a daprd shutdown.
		output, err := cmdRun("", "--dapr-http-port", "9999", "--", "bash", "-c", "curl -v -X POST http://localhost:9999/v1.0/shutdown; sleep 10; exit 1")
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully", "App should be shutdown before it has a chance to return non-zero")
		assert.Contains(t, output, "Exited Dapr successfully")
		assert.NotContains(t, output, "Could not update sidecar metadata for cliPID")
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
		assert.NotContains(t, output, "Could not update sidecar metadata for cliPID")
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
		assert.Contains(t, output, "level=info msg=\"HTTP API Called\" app_id=enableApiLogging_info")
		assert.Contains(t, output, "method=\"PUT /v1.0/metadata/appCommand\"")
		assert.Contains(t, output, "method=\"PUT /v1.0/metadata/cliPID\"")
		assert.Contains(t, output, "method=\"PUT /v1.0/metadata/appPID\"")
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
		assert.NotContains(t, output, "level=info msg=\"HTTP API Called\" app_id=enableApiLogging_info")
		assert.NotContains(t, output, "method=\"PUT /v1.0/metadata/appCommand\"")
		assert.NotContains(t, output, "method=\"PUT /v1.0/metadata/cliPID\"")
		assert.NotContains(t, output, "method=\"PUT /v1.0/metadata/appPID\"")
	})

	t.Run(fmt.Sprintf("check enableAPILogging with obfuscation through dapr config file"), func(t *testing.T) {
		args := []string{
			"--app-id", "enableApiLogging_info",
			"--config", "../testdata/config.yaml",
			"--", "bash", "-c", "echo 'test'",
		}

		output, err := cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
		assert.Contains(t, output, "level=info msg=\"HTTP API Called\" app_id=enableApiLogging_info")
		assert.Contains(t, output, "method=\"PUT /v1.0/metadata/{key}\"")
		assert.Contains(t, output, "method=\"PUT /v1.0/metadata/{key}\"")
		assert.Contains(t, output, "method=\"PUT /v1.0/metadata/{key}\"")
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

	t.Run("check run with nonexistent resources-path", func(t *testing.T) {
		args := []string{
			"--app-id", "testapp",
			"--resources-path", "../testdata/nonexistentdir",
			"--", "bash", "-c", "echo 'test'",
		}
		output, err := cmdRun("", args...)
		t.Log(output)
		require.Error(t, err, "run did not fail")
	})

	t.Run("check run with resources-path", func(t *testing.T) {
		args := []string{
			"--app-id", "testapp",
			"--resources-path", "../testdata/resources",
			"--", "bash", "-c", "echo 'test'",
		}
		output, err := cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Component loaded: test-statestore (state.in-memory/v1)")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
	})

	t.Run("check run with multiple resources-path", func(t *testing.T) {
		args := []string{
			"--app-id", "testapp",
			"--resources-path", "../testdata/resources",
			"--resources-path", "../testdata/additional_resources",
			"--", "bash", "-c", "echo 'test'",
		}
		output, err := cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Component loaded: test-statestore (state.in-memory/v1)")
		assert.Contains(t, output, "Component loaded: test-statestore-additional-resource (state.in-memory/v1)")
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
