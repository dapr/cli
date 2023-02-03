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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dapr/cli/tests/e2e/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStandaloneInitRunUninstallNonDefaultDaprPath covers init, version, run and uninstall with --dapr-path flag.
func TestStandaloneInitRunUninstallNonDefaultDaprPath(t *testing.T) {
	// Ensure a clean environment
	must(t, cmdUninstall, "failed to uninstall Dapr")
	t.Run("run with dapr path flag", func(t *testing.T) {
		daprPath, err := os.MkdirTemp("", "dapr-e2e-run-with-flag-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath) // clean up

		daprRuntimeVersion, _ := common.GetVersionsFromEnv(t, false)
		output, err := cmdInit("--runtime-version", daprRuntimeVersion, "--dapr-path", daprPath)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		// check version
		output, err = cmdVersion("", "--dapr-path", daprPath)
		t.Log(output)
		require.NoError(t, err, "dapr version failed")
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 2, "expected at least 2 fields in components outptu")
		assert.Contains(t, lines[0], "CLI version")
		assert.Contains(t, lines[0], "edge")
		assert.Contains(t, lines[1], "Runtime version")
		assert.Contains(t, lines[1], daprRuntimeVersion)

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

		// Uninstall Dapr at the end of the test since it's being installed in a non-default location.
		must(t, cmdUninstall, "failed to uninstall Dapr from custom path flag", "--dapr-path", daprPath)
		customDaprPath := filepath.Join(daprPath, ".dapr")
		assert.NoDirExists(t, customDaprPath)
		assert.DirExists(t, daprPath)
		// Check the directory is empty.
		f, err := os.ReadDir(daprPath)
		assert.NoError(t, err)
		assert.Len(t, f, 0)
	})

	t.Run("run with env var", func(t *testing.T) {
		daprPath, err := os.MkdirTemp("", "dapr-e2e-run-with-env-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath) // clean up

		t.Setenv("DAPR_PATH", daprPath)

		daprRuntimeVersion, _ := common.GetVersionsFromEnv(t, false)

		output, err := cmdInit("--runtime-version", daprRuntimeVersion)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		// check version
		output, err = cmdVersion("", "--dapr-path", daprPath)
		t.Log(output)
		require.NoError(t, err, "dapr version failed")
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 2, "expected at least 2 fields in components outptu")
		assert.Contains(t, lines[0], "CLI version")
		assert.Contains(t, lines[0], "edge")
		assert.Contains(t, lines[1], "Runtime version")
		assert.Contains(t, lines[1], daprRuntimeVersion)

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

		// Uninstall Dapr at the end of the test since it's being installed in a non-default location.
		must(t, cmdUninstall, "failed to uninstall Dapr from custom env var path")
		customDaprPath := filepath.Join(daprPath, ".dapr")
		assert.NoDirExists(t, customDaprPath)
		assert.DirExists(t, daprPath)
		// Check the directory is empty.
		f, err := os.ReadDir(daprPath)
		assert.NoError(t, err)
		assert.Len(t, f, 0)
	})

	t.Run("run with both flag and env var", func(t *testing.T) {
		daprPathEnv, err := os.MkdirTemp("", "dapr-e2e-run-with-envflag-1-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPathEnv) // clean up

		daprPathFlag, err := os.MkdirTemp("", "dapr-e2e-run-with-envflag-2-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPathFlag) // clean up

		t.Setenv("DAPR_PATH", daprPathEnv)

		daprRuntimeVersion, _ := common.GetVersionsFromEnv(t, false)

		output, err := cmdInit("--runtime-version", daprRuntimeVersion, "--dapr-path", daprPathFlag)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		// check version
		output, err = cmdVersion("", "--dapr-path", daprPathFlag)
		t.Log(output)
		require.NoError(t, err, "dapr version failed")
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 2, "expected at least 2 fields in components outptu")
		assert.Contains(t, lines[0], "CLI version")
		assert.Contains(t, lines[0], "edge")
		assert.Contains(t, lines[1], "Runtime version")
		assert.Contains(t, lines[1], daprRuntimeVersion)

		args := []string{
			"--dapr-path", daprPathFlag,
			"--app-id", "run_with_dapr_path_flag",
			"--", "bash", "-c", "echo 'test'",
		}

		flagDaprdBinPath := filepath.Join(daprPathFlag, ".dapr", "bin", "daprd")
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
		assert.NoDirExists(t, defaultDaprPath)

		envDaprBinPath := filepath.Join(daprPathEnv, ".dapr", "bin")
		assert.NoDirExists(t, envDaprBinPath)

		// Uninstall Dapr at the end of the test since it's being installed in a non-default location.
		must(t, cmdUninstall, "failed to uninstall Dapr from custom path flag", "--dapr-path", daprPathFlag)
		customDaprPath := filepath.Join(daprPathFlag, ".dapr")
		assert.NoDirExists(t, customDaprPath)
		assert.DirExists(t, daprPathFlag)
		// Check the directory is empty.
		f, err := os.ReadDir(daprPathFlag)
		assert.NoError(t, err)
		assert.Len(t, f, 0)
	})
}
