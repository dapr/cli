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
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dapr/cli/tests/e2e/common"

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
		assert.Contains(t, output, "level=info msg=\"HTTP API Called\" app_id=enableApiLogging_info")
		assert.Contains(t, output, "method=\"PUT /v1.0/metadata/{key}\"")
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

func TestStandaloneRunNonDefaultDaprPath(t *testing.T) {
	// Uninstall Dapr at the end of the test since it's being installed in a non-default location.
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	t.Run("run with flag", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		daprPath, err := os.MkdirTemp("", "dapr-e2e-run-with-flag-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath) // clean up

		daprRuntimeVersion, _ := common.GetVersionsFromEnv(t, false)
		output, err := cmdInit("--runtime-version", daprRuntimeVersion, "--dapr-path", daprPath)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		args := []string{
			"--dapr-path", daprPath,
			"--app-id", "run_with_dapr_path_flag",
			"--", "bash", "-c", "echo 'test'",
		}

		output, err = cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		defaultDaprPath := filepath.Join(homeDir, ".dapr")
		assert.NoFileExists(t, defaultDaprPath)
	})

	t.Run("run with env var", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		daprPath, err := os.MkdirTemp("", "dapr-e2e-run-with-env-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath) // clean up

		t.Setenv("DAPR_PATH", daprPath)

		daprRuntimeVersion, _ := common.GetVersionsFromEnv(t, false)

		output, err := cmdInit("--runtime-version", daprRuntimeVersion)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		args := []string{
			"--app-id", "run_with_dapr_path_flag",
			"--", "bash", "-c", "echo 'test'",
		}

		output, err = cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		defaultDaprPath := filepath.Join(homeDir, ".dapr")
		assert.NoFileExists(t, defaultDaprPath)
	})

	t.Run("run with both flag and env var", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		daprPathForEnv, err := os.MkdirTemp("", "dapr-e2e-run-with-envflag-1-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPathForEnv) // clean up

		daprPathForFlag, err := os.MkdirTemp("", "dapr-e2e-run-with-envflag-2-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPathForFlag) // clean up

		t.Setenv("DAPR_PATH", daprPathForEnv)

		daprRuntimeVersion, _ := common.GetVersionsFromEnv(t, false)

		output, err := cmdInit("--runtime-version", daprRuntimeVersion, "--dapr-path", daprPathForFlag)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		args := []string{
			"--dapr-path", daprPathForFlag,
			"--app-id", "run_with_dapr_path_flag",
			"--", "bash", "-c", "echo 'test'",
		}

		flagDaprdBinPath := filepath.Join(daprPathForFlag, "bin", "daprd")
		if runtime.GOOS == "windows" {
			flagDaprdBinPath += ".exe"
		}
		assert.FileExists(t, flagDaprdBinPath)

		output, err = cmdRun("", args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		defaultDaprPath := filepath.Join(homeDir, ".dapr")
		assert.NoFileExists(t, defaultDaprPath)

		envDaprBinPath := filepath.Join(daprPathForEnv, "bin")
		assert.NoFileExists(t, envDaprBinPath)
	})
}
