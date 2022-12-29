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
	"fmt"
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

// Tests precedence for --components-path and --resources-path flags.
func TestResourcesLoadPrecedence(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err, "cannot get user home directory")
	defaultComponentsDirPath = filepath.Join(homeDir, ".dapr", utils.DefaultComponentsDirName)
	defaultResourcesDirPath = filepath.Join(homeDir, ".dapr", utils.DefaultResourcesDirName)

	t.Run("without pre-existing components directory", func(t *testing.T) {
		// Ensure a clean environment.
		must(t, cmdUninstallAll, "failed to uninstall Dapr")

		// install dapr -> installs dapr, creates resources dir and symlink components dir.
		ensureDaprInstallation(t)

		// check dapr run -> should not load a in-memory statestore component "test-statestore".
		testDaprRunOutput(t, false)

		// copy an in-memomy state store component to the resources directory.
		copyInMemStateStore(t, defaultResourcesDirPath)

		// check dapr run -> should load in-memory component "test-statestore" from resources directory.
		testDaprRunOutput(t, true)

		// dapr run with --components-path flag -> should also load the in-memory component because "components" directory is symlinked to "resources" directory.
		testDaprRunOutput(t, true, "--components-path", defaultComponentsDirPath)

		// dapr run with --resources-path flag -> should also load the in-memory component.
		testDaprRunOutput(t, true, "--resources-path", defaultResourcesDirPath)

		// dapr run with both flags --resources-path and --components-path.
		args := []string{
			"--components-path", defaultComponentsDirPath,
			"--resources-path", defaultResourcesDirPath,
		}
		testDaprRunOutput(t, true, args...)
	})

	t.Run("with pre-existing components directory", func(t *testing.T) {
		// Ensure a clean environment.
		must(t, cmdUninstallAll, "failed to uninstall Dapr")

		// install dapr -> installs dapr, creates resources dir and symlink components dir.
		ensureDaprInstallation(t)

		// test setup -> remove created symlink and rename resources directory to components.
		prepareComponentsDir(t)

		// copy an in-memomy state store component to the components directory.
		copyInMemStateStore(t, defaultComponentsDirPath)

		// uninstall without removing the components directory.
		must(t, cmdUninstall, "failed to uninstall Dapr")

		// install dapr -> installs dapr. It does following -
		// 1) creates resources directory. 2)copy resources from components to resources directory.
		// 3) delete components directory. 4) creates symlink components for resources directory.
		ensureDaprInstallation(t)

		// check dapr run -> should load the in-memory statestore component "test-statestore".
		testDaprRunOutput(t, true)

		// dapr run with --components-path flag -> should also load the in-memory component because "components" directory is symlinked to "resources" directory.
		testDaprRunOutput(t, true, "--components-path", defaultComponentsDirPath)

		// dapr run with --resources-path flag -> should also load the in-memory component.
		testDaprRunOutput(t, true, "--resources-path", defaultResourcesDirPath)
	})

	t.Run("add resources to components directory post dapr install", func(t *testing.T) {
		// Ensure a clean environment.
		must(t, cmdUninstallAll, "failed to uninstall Dapr")

		// install dapr -> installs dapr, creates resources dir and symlink components dir.
		ensureDaprInstallation(t)

		// check dapr run -> should not load a in-memory statestore component "test-statestore".
		testDaprRunOutput(t, false)

		// copy an in-memomy state store component to the components directory.
		copyInMemStateStore(t, defaultComponentsDirPath)

		// check dapr run -> should load in-memory component "test-statestore" from resources directory.
		testDaprRunOutput(t, true)

		// dapr run with --components-path flag -> should also load the in-memory component because "components" directory is symlinked to "resources" directory.
		testDaprRunOutput(t, true, "--components-path", defaultComponentsDirPath)

		// dapr run with --resources-path flag -> should also load the in-memory component.
		testDaprRunOutput(t, true, "--resources-path", defaultResourcesDirPath)
	})
}

func prepareComponentsDir(t *testing.T) {
	// remove symlink.
	err := os.Remove(defaultComponentsDirPath)
	assert.NoError(t, err, fmt.Sprintf("cannot remove symlink %q", defaultComponentsDirPath))

	// rename resources directory to components.
	err = os.Rename(defaultResourcesDirPath, defaultComponentsDirPath)
	assert.NoError(t, err, fmt.Sprintf("cannot rename %q to %q", defaultResourcesDirPath, defaultComponentsDirPath))
}

func copyInMemStateStore(t *testing.T, targetDirPath string) {
	filePath := filepath.Join("../testdata/resources", "test-statestore.yaml")
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err, "cannot read testdata/resources/test-statestore.yaml file")
	err = os.WriteFile(filepath.Join(targetDirPath, "test-statestore.yaml"), content, 0644)
	assert.NoError(t, err, fmt.Sprintf("cannot write testdata/resources/test-statestore.yaml file to %q directory", targetDirPath))
}

func testDaprRunOutput(t *testing.T, inMemoryCompPresent bool, flags ...string) {
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
