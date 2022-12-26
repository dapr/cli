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
package standalone_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dapr/cli/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// DefaultComponentsDirPath is the default components directory path.
	defaultComponentsDirPath = ""
	defaultResourcesDirPath  = ""
)

// It covers the test flow for scenario when user does: (1) dapr uninstall (2) upgrades dapr cli (3) dapr init (4) dapr run.
func TestCompToResrcDirMig(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err, "cannot get user home directory")
	defaultComponentsDirPath = filepath.Join(homeDir, ".dapr", utils.DefaultComponentsDirName)
	defaultResourcesDirPath = filepath.Join(homeDir, ".dapr", utils.DefaultResourcesDirName)
	// Ensure a clean environment.
	must(t, cmdUninstall, "failed to uninstall Dapr")

	// install dapr -> installs dapr, creates resources dir and symlink components dir.
	ensureDaprInstallation(t)

	// check dapr run -> should not load in-memory component.
	checkDaprRunPrecedenceTest(t, false)

	// copy a in memomy state store component to resources dir.
	copyInMemStateStore(t)

	// check dapr run -> should load in-memory component.
	checkDaprRunPrecedenceTest(t, true)

	// dapr run with --components-path flag -> should load in-memory component.
	checkDaprRunPrecedenceTest(t, true, "--components-path", defaultComponentsDirPath)

	// dapr run with --resources-path flag -> should load in-memory component.
	checkDaprRunPrecedenceTest(t, true, "--resources-path", defaultResourcesDirPath)
}

func copyInMemStateStore(t *testing.T) {
	filePath := filepath.Join("../testdata/resources", "test-statestore.yaml")
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err, "cannot read testdata/resources/test-statestore.yaml file")
	err = os.WriteFile(filepath.Join(defaultResourcesDirPath, "test-statestore.yaml"), content, 0644)
	assert.NoError(t, err, "cannot write testdata/resources/test-statestore.yaml file to resources directory")
}

func checkDaprRunPrecedenceTest(t *testing.T, inMemoryCompPresent bool, flags ...string) {
	args := []string{
		"--app-id", "testapp",
		"--", "bash", "-c", "echo 'test'",
	}
	args = append(args, flags...)
	output, err := cmdRun("", args...)
	t.Log(output)
	require.NoError(t, err, "run failed")
	if inMemoryCompPresent {
		assert.Contains(t, output, "component loaded. name: test-statestore, type: state.in-memory/v1")
	} else {
		assert.NotContains(t, output, "component loaded. name: test-statestore, type: state.in-memory/v1")
	}
	assert.Contains(t, output, "Exited App successfully")
	assert.Contains(t, output, "Exited Dapr successfully")
}
