//go:build e2e && !template

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
	"runtime"
	"strings"
	"testing"

	"github.com/dapr/cli/tests/e2e/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandaloneRun(t *testing.T) {
	ensureDaprInstallation(t)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	if isSlimMode() {
		output, err := cmdProcess(ctx, "placement", t.Log, "--metrics-port", "9091", "--healthz-port", "8081")
		require.NoError(t, err)
		t.Log(output)

		if common.GetRuntimeVersion(t, false).GreaterThan(common.VersionWithScheduler) {
			output, err = cmdProcess(ctx, "scheduler", t.Log, "--metrics-port", "9092", "--healthz-port", "8082")
			require.NoError(t, err)
			t.Log(output)
		}
	}
	t.Cleanup(func() {
		// remove dapr installation after all tests in this function.
		must(t, cmdUninstall, "failed to uninstall Dapr")
		// Call cancelFunc to stop the processes
		cancelFunc()
	})
	for _, path := range getSocketCases() {
		t.Run(fmt.Sprintf("normal exit, socket: %s", path), func(t *testing.T) {
			output, err := cmdRun(path, append([]string{"--"}, echoTestAppArgs()...)...)
			t.Log(output)
			require.NoError(t, err, "run failed")
			assert.Contains(t, output, "Exited App successfully")
			assert.Contains(t, output, "Exited Dapr successfully")
			assert.NotContains(t, output, "Could not update sidecar metadata for cliPID")
		})

		t.Run(fmt.Sprintf("error exit, socket: %s", path), func(t *testing.T) {
			args := []string{"--"}
			if runtime.GOOS == "windows" {
				args = append(args, "cmd", "/c", "echo test & exit /b 1")
			} else {
				args = append(args, "bash", "-c", "echo 'test'; exit 1")
			}
			output, err := cmdRun(path, args...)
			t.Log(output)
			require.Error(t, err, "run failed")
			// CLI may print "exit status 1" or "1"
			assert.True(t,
				strings.Contains(output, "The App process exited with error code: exit status 1") ||
					strings.Contains(output, "The App process exited with error code: 1") ||
					strings.Contains(output, "The App process exited with error: exit status 1") ||
					strings.Contains(output, "The App process exited with error: 1"),
				"expected app error exit message in output: %s", output)
			assert.Contains(t, output, "Exited Dapr successfully")
			assert.NotContains(t, output, "Could not update sidecar metadata for cliPID")
		})

		t.Run("Use internal gRPC port if specified", func(t *testing.T) {
			output, err := cmdRun(path, append([]string{"--dapr-internal-grpc-port", "9999", "--"}, echoTestAppArgs()...)...)
			t.Log(output)
			require.NoError(t, err, "run failed")
			if common.GetRuntimeVersion(t, false).GreaterThan(common.VersionWithScheduler) {
				assert.Contains(t, output, "Internal gRPC server is running on :9999")
			} else {
				assert.Contains(t, output, "Internal gRPC server is running on port 9999")
			}
			assert.Contains(t, output, "Exited App successfully")
			assert.Contains(t, output, "Exited Dapr successfully")
			assert.NotContains(t, output, "Could not update sidecar metadata for cliPID")
		})
	}

	t.Run("API shutdown without socket", func(t *testing.T) {
		// Test that the CLI exits on a daprd shutdown.
		args := []string{"--dapr-http-port", "9999", "--"}
		if runtime.GOOS == "windows" {
			args = append(args, "cmd", "/c", "curl -v -X POST http://localhost:9999/v1.0/shutdown && timeout /t 10 && exit 1")
		} else {
			args = append(args, "bash", "-c", "curl -v -X POST http://localhost:9999/v1.0/shutdown; sleep 10; exit 1")
		}
		output, err := cmdRun("", args...)
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
		args := []string{"--app-id", "testapp", "--"}
		args = append(args, "bash", "-c", "curl --unix-socket /tmp/dapr-testapp-http.socket -v -X POST http://unix/v1.0/shutdown; sleep 10; exit 1")
		output, err := cmdRun("/tmp", args...)
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
			"--",
		}
		args = append(args, echoTestAppArgs()...)

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
			"--",
		}
		args = append(args, echoTestAppArgs()...)

		output, err := cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
		assert.NotContains(t, output, "level=info msg=\"HTTP API Called\" app_id=enableApiLogging_info")
		assert.NotContains(t, output, "method=PutMetadata")
		assert.NotContains(t, output, "Updating metadata for appCommand: ")
		assert.NotContains(t, output, "Updating metadata for cliPID: ")
	})

	t.Run(fmt.Sprintf("check enableAPILogging with obfuscation through dapr config file"), func(t *testing.T) {
		args := []string{
			"--app-id", "enableApiLogging_info",
			"--config", "../testdata/config.yaml",
			"--",
		}
		args = append(args, echoTestAppArgs()...)

		output, err := cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
		assert.Contains(t, output, "level=info msg=\"HTTP API Called\" app_id=enableApiLogging_info")
		assert.Contains(t, output, "method=PutMetadata")
		assert.Contains(t, output, "Updating metadata for appPID: ")
	})

	t.Run(fmt.Sprintf("check run with log JSON enabled"), func(t *testing.T) {
		args := []string{
			"--app-id", "logjson",
			"--log-as-json",
			"--",
		}
		args = append(args, echoTestAppArgs()...)

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
			"--",
		}
		args = append(args, echoTestAppArgs()...)

		output, err := cmdRun("", args...)
		t.Log(output)
		require.Error(t, err, "run did not fail")
	})

	t.Run("check run with resources-path", func(t *testing.T) {
		args := []string{
			"--app-id", "testapp",
			"--resources-path", "../testdata/resources",
			"--",
		}
		args = append(args, echoTestAppArgs()...)

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
			"--",
		}
		args = append(args, echoTestAppArgs()...)
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
