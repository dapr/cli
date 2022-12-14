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

// TODO: Remove the test file when `--components-path` flag is removed.

// This file contains tests for the migration of components directory to resources directory.
// It covers the test flow for scenario when user does: (1) dapr uninstall (2) upgrades dapr cli (3) dapr init (4) dapr run.
package standalone_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dapr/cli/tests/e2e/common"
	"github.com/dapr/cli/tests/e2e/spawn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// DefaultComponentsDirPath is the default components directory path.
	defaultComponentsDirPath = ""
	defaultResourcesDirPath  = ""
)

func TestCompToResrcDirMig(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err, "cannot get user home directory")
	defaultComponentsDirPath = filepath.Join(homeDir, ".dapr", "components")
	defaultResourcesDirPath = filepath.Join(homeDir, ".dapr", "resources")
	// Ensure a clean environment.
	must(t, cmdUninstall, "failed to uninstall Dapr")

	// install dapr.
	ensureDaprInstallation(t)

	// rename resources to components dir.
	renameResourcesDir(t)

	// dapr run should work with only components dir.
	// 2nd paramter is true to indicate that only components dir is present.
	checkDaprRunPrecedenceTest(t, true)

	// dapr uninstall without --all flag should work.
	uninstallWithoutAllFlag(t)

	// dapr init should duplicate files from components dir to resources dir.
	initTest(t)

	// copy a in memomy state store component to resources dir.
	copyInMemStateStore(t)

	// check dapr run precedence order, resources dir 1st then components dir.
	// 2nd paramter is false to indicate that both components and resources dir are present.
	checkDaprRunPrecedenceTest(t, false)
}

func copyInMemStateStore(t *testing.T) {
	filePath := filepath.Join("../testdata/resources", "test-statestore.yaml")
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err, "cannot read testdata/resources/test-statestore.yaml file")
	err = os.WriteFile(filepath.Join(defaultResourcesDirPath, "test-statestore.yaml"), content, 0644)
	assert.NoError(t, err, "cannot write testdata/resources/test-statestore.yaml file to resources directory")
}

func initTest(t *testing.T) {
	daprRuntimeVersion, _ := common.GetVersionsFromEnv(t)

	output, err := cmdInit(daprRuntimeVersion)
	t.Log(output)
	require.NoError(t, err, "init failed")
	assert.Contains(t, output, "Success! Dapr is up and running.")

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "failed to get user home directory")

	daprPath := filepath.Join(homeDir, ".dapr")
	require.DirExists(t, daprPath, "Directory %s does not exist", daprPath)
	require.DirExists(t, defaultComponentsDirPath, "Components dir does not exist")
	require.DirExists(t, defaultResourcesDirPath, "Resources dir does not exist")
}

func checkDaprRunPrecedenceTest(t *testing.T, onlyCompDirPresent bool) {
	args := []string{
		"--app-id", "testapp",
		"--", "bash", "-c", "echo 'test'",
	}
	output, err := cmdRun("", args...)
	t.Log(output)
	require.NoError(t, err, "run failed")
	if onlyCompDirPresent {
		assert.NotContains(t, output, "component loaded. name: test-statestore, type: state.in-memory/v1")
	} else {
		assert.Contains(t, output, "component loaded. name: test-statestore, type: state.in-memory/v1")
	}
	assert.Contains(t, output, "Exited App successfully")
	assert.Contains(t, output, "Exited Dapr successfully")
}

func uninstallWithoutAllFlag(t *testing.T) {
	uninstallArgs := []string{"uninstall"}
	daprContainerRuntime := containerRuntime()

	// Add --container-runtime flag only if daprContainerRuntime is not empty, or overridden via args.
	// This is only valid for non-slim mode.
	if !isSlimMode() && daprContainerRuntime != "" {
		uninstallArgs = append(uninstallArgs, "--container-runtime", daprContainerRuntime)
	}
	_, error := spawn.Command(common.GetDaprPath(), uninstallArgs...)
	if error != nil {
		assert.NoError(t, error, "failed to uninstall Dapr")
	}
}

func renameResourcesDir(t *testing.T) {
	err := os.Rename(defaultResourcesDirPath, defaultComponentsDirPath)
	if err != nil {
		mesg := fmt.Sprintf("pre-req to TestCompToResrcDirMig failed. error renaming components dir to resources dir: %s", err)
		assert.NoError(t, err, mesg)
	}
}
