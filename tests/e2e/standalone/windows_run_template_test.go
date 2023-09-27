//go:build windows && (e2e || template)
// +build windows
// +build e2e template

/*
Copyright 2023 The Dapr Authors
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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type AppTestOutput struct {
	appID                    string
	appLogContents           []string
	daprdLogContent          []string
	baseLogDirPath           string
	appLogDoesNotExist       bool
	daprdLogFileDoesNotExist bool
}

func TestRunWithTemplateFile(t *testing.T) {
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		// remove dapr installation after all tests in this function.
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})
	// These tests are dependent on run template files in ../testdata/run-template-files folder.
	t.Run("valid template file", func(t *testing.T) {
		runFilePath := "../testdata/run-template-files/dapr.yaml"
		go startAppsWithValidRunTemplate(t, runFilePath)
		time.Sleep(10 * time.Second)
		output, err := cmdStopWithRunTemplate(runFilePath)
		assert.NoError(t, err, "failed to stop apps started with run template")
		assert.Contains(t, output, "Dapr and app processes stopped successfully")
		time.Sleep(5 * time.Second)
	})

	t.Run("valid template file with App output written to only file", func(t *testing.T) {
		runFilePath := "../testdata/run-template-files/app_output_to_file_and_console.yaml"
		go startAppsWithAppLogDestFile(t, runFilePath)
		time.Sleep(10 * time.Second)
		output, err := cmdStopWithRunTemplate(runFilePath)
		assert.NoError(t, err, "failed to stop apps started with run template")
		assert.Contains(t, output, "Dapr and app processes stopped successfully")
		time.Sleep(5 * time.Second)
	})

	t.Run("valid template file with App output written to only console", func(t *testing.T) {
		runFilePath := "../testdata/run-template-files/app_output_to_only_console.yaml"
		go startAppsWithAppLogDestConsole(t, runFilePath)
		time.Sleep(10 * time.Second)
		output, err := cmdStopWithRunTemplate(runFilePath)
		assert.NoError(t, err, "failed to stop apps started with run template")
		assert.Contains(t, output, "Dapr and app processes stopped successfully")
		time.Sleep(5 * time.Second)
	})
}

func startAppsWithValidRunTemplate(t *testing.T, file string) {
	args := []string{
		"-f", file,
	}
	output, err := cmdRun("", args...)
	t.Logf(output)
	require.NoError(t, err, "run failed")
	lines := strings.Split(output, "\n")
	assert.GreaterOrEqual(t, len(lines), 6, "expected at least 6 lines in output of starting two apps")
	assert.Contains(t, lines[0], "Validating config and starting app \"processor\"")
	assert.Contains(t, lines[1], "Started Dapr with app id \"processor\". HTTP Port: 3510.")
	assert.Contains(t, lines[2], "Writing log files to directory")
	assert.Contains(t, lines[2], "tests\\apps\\processor\\.dapr\\logs")
	assert.Contains(t, lines[3], "Validating config and starting app \"emit-metrics\"")
	assert.Contains(t, lines[4], "Started Dapr with app id \"emit-metrics\". HTTP Port: 3511.")
	assert.Contains(t, lines[5], "Writing log files to directory")
	assert.Contains(t, lines[5], "tests\\apps\\emit-metrics\\.dapr\\logs")
}

func startAppsWithAppLogDestFile(t *testing.T, file string) {
	args := []string{
		"-f", file,
	}
	output, err := cmdRun("", args...)
	t.Logf(output)
	require.NoError(t, err, "run failed")

	// App logs for processor app should not be printed to console and only written to file.
	assert.NotContains(t, output, "== APP - processor")

	// Daprd logs for processor app should only be printed to console and not written to file.
	assert.Contains(t, output, "msg=\"All outstanding components processed\" app_id=processor")

	// App logs for emit-metrics app should be printed to console and written to file.
	assert.Contains(t, output, "== APP - emit-metrics")

	// Daprd logs for emit-metrics app should only be written to file.
	assert.NotContains(t, output, "msg=\"All outstanding components processed\" app_id=emit-metrics")

	assert.Contains(t, output, "Received signal to stop Dapr and app processes. Shutting down Dapr and app processes.")

}

func startAppsWithAppLogDestConsole(t *testing.T, file string) {
	args := []string{
		"-f", file,
	}
	output, err := cmdRun("", args...)
	t.Logf(output)
	require.NoError(t, err, "run failed")

	// App logs for processor app should be printed to console.
	assert.Contains(t, output, "== APP - processor")

	// Daprd logs for processor app should only be written to file.
	assert.NotContains(t, output, "msg=\"All outstanding components processed\" app_id=processor")

	// App logs for emit-metrics app should be printed to console.
	assert.Contains(t, output, "== APP - emit-metrics")

	// Daprd logs for emit-metrics app should only be written to file.
	assert.NotContains(t, output, "msg=\"All outstanding components processed\" app_id=emit-metrics")

	assert.Contains(t, output, "Received signal to stop Dapr and app processes. Shutting down Dapr and app processes.")

}
