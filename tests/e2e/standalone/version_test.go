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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dapr/cli/tests/e2e/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandaloneVersion(t *testing.T) {
	ensureDaprInstallation(t)

	t.Run("version", func(t *testing.T) {
		output, err := cmdVersion("")
		t.Log(output)
		require.NoError(t, err, "dapr version failed")
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 2, "expected at least 2 fields in components outptu")
		assert.Contains(t, lines[0], "CLI version")
		assert.Contains(t, lines[1], "Runtime version")
	})

	t.Run("version json", func(t *testing.T) {
		output, err := cmdVersion("json")
		t.Log(output)
		require.NoError(t, err, "dapr version failed")
		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		assert.NoError(t, err, "output was not valid JSON")
	})
}

func TestStandaloneVersionNonDefaultDaprPath(t *testing.T) {
	t.Run("version with flag", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		daprPath, err := os.MkdirTemp("", "dapr-e2e-ver-with-flag-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath) // clean up

		daprRuntimeVersion, _ := common.GetVersionsFromEnv(t)
		output, err := cmdInit(daprRuntimeVersion, "--dapr-path", daprPath)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		output, err = cmdVersion("", "--dapr-path", daprPath)
		t.Log(output)
		require.NoError(t, err, "dapr version failed")
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 2, "expected at least 2 fields in components outptu")
		assert.Contains(t, lines[0], "CLI version")
		assert.Contains(t, lines[1], "Runtime version")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		defaultDaprPath := filepath.Join(homeDir, ".dapr")
		assert.NoFileExists(t, defaultDaprPath)
	})

	t.Run("version with env var", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		daprPath, err := os.MkdirTemp("", "dapr-e2e-ver-with-env-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath) // clean up

		os.Setenv("DAPR_PATH", daprPath)
		defer os.Unsetenv("DAPR_PATH")

		daprRuntimeVersion, _ := common.GetVersionsFromEnv(t)
		output, err := cmdInit(daprRuntimeVersion)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		output, err = cmdVersion("")
		t.Log(output)
		require.NoError(t, err, "dapr version failed")
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 2, "expected at least 2 fields in components outptu")
		assert.Contains(t, lines[0], "CLI version")
		assert.Contains(t, lines[1], "Runtime version")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		defaultDaprPath := filepath.Join(homeDir, ".dapr")
		assert.NoFileExists(t, defaultDaprPath)
	})

	t.Run("version with both flag and env var", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		daprPath1, err := os.MkdirTemp("", "dapr-e2e-ver-with-both-flag-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath1) // clean up

		daprPath2, err := os.MkdirTemp("", "dapr-e2e-ver-with-both-env-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath2) // clean up

		os.Setenv("DAPR_PATH", daprPath2)
		defer os.Unsetenv("DAPR_PATH")

		daprRuntimeVersion, _ := common.GetVersionsFromEnv(t)
		output, err := cmdInit(daprRuntimeVersion, "--dapr-path", daprPath1)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		output, err = cmdVersion("", "--dapr-path", daprPath1)
		t.Log(output)
		require.NoError(t, err, "dapr version failed")
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 2, "expected at least 2 fields in components outptu")
		assert.Contains(t, lines[0], "CLI version")
		assert.Contains(t, lines[1], "Runtime version")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		defaultDaprPath := filepath.Join(homeDir, ".dapr")
		assert.NoFileExists(t, defaultDaprPath)

		envDaprBinPath := filepath.Join(daprPath2, "bin")
		assert.NoFileExists(t, envDaprBinPath)
	})
}
