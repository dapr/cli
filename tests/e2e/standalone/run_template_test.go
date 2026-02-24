//go:build !windows && (e2e || template)

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
	"fmt"
	"io/ioutil"
	"path/filepath"
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
	daprdLogPollTimeout      time.Duration
}

func TestRunWithTemplateFile(t *testing.T) {
	cmdUninstall()
	cleanUpLogs()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		// remove dapr installation after all tests in this function.
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})
	// These tests are dependent on run template files in ../testdata/run-template-files folder.

	t.Run("invalid template file wrong emit metrics app run", func(t *testing.T) {
		runFilePath := "../testdata/run-template-files/wrong_emit_metrics_app_dapr.yaml"
		t.Cleanup(func() {
			// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}

		outputCh := make(chan string)
		go func() {
			output, _ := cmdRun("", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		time.Sleep(time.Second * 10)
		cmdStopWithRunTemplate(runFilePath)
		var output string
		select {
		case output = <-outputCh:
		case <-time.After(25 * time.Second):
			t.Fatal("timed out waiting for run command to finish")
		}

		// Deterministic output for template file, so we can assert line by line
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 4, "expected at least 4 lines in output of starting two apps")
		assert.Contains(t, lines[1], "Started Dapr with app id \"processor\". HTTP Port: 3510.")
		assert.Contains(t, lines[2], "Writing log files to directory")
		assert.Contains(t, lines[2], "tests/apps/processor/.dapr/logs")
		assert.Contains(t, lines[4], "Started Dapr with app id \"emit-metrics\". HTTP Port: 3511.")
		assert.Contains(t, lines[5], "Writing log files to directory")
		assert.Contains(t, lines[5], "tests/apps/emit-metrics/.dapr/logs")
		assert.Contains(t, output, "Received signal to stop Dapr and app processes. Shutting down Dapr and app processes.")
		appTestOutput := AppTestOutput{
			appID:          "processor",
			baseLogDirPath: "../../apps/processor/.dapr/logs",
			daprdLogContent: []string{
				"HTTP server is running on port 3510",
				"You're up and running! Dapr logs will appear here.",
			},
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
		appTestOutput = AppTestOutput{
			appID:          "emit-metrics",
			baseLogDirPath: "../../apps/emit-metrics/.dapr/logs",
			appLogContents: []string{
				"stat wrongappname.go: no such file or directory",
				"The App process exited with error code: exit status 1",
			},
			daprdLogContent: []string{
				"termination signal received: shutting down",
				"Exited Dapr successfully",
			},
			daprdLogPollTimeout: 15 * time.Second,
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
	})

	t.Run("valid template file", func(t *testing.T) {
		cmdUninstall()
		ensureDaprInstallation(t)

		runFilePath := "../testdata/run-template-files/dapr.yaml"
		t.Cleanup(func() {
			// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}

		outputCh := make(chan string)
		go func() {
			output, _ := cmdRun("", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		time.Sleep(time.Second * 10)
		cmdStopWithRunTemplate(runFilePath)
		var output string
		select {
		case output = <-outputCh:
		case <-time.After(time.Second * 10):
			t.Fatal("timed out waiting for run command to finish")
		}

		// Deterministic output for template file, so we can assert line by line
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 6, "expected at least 6 lines in output of starting two apps")
		assert.Contains(t, lines[0], "Validating config and starting app \"processor\"")
		assert.Contains(t, lines[1], "Started Dapr with app id \"processor\". HTTP Port: 3510.")
		assert.Contains(t, lines[2], "Writing log files to directory")
		assert.Contains(t, lines[2], "tests/apps/processor/.dapr/logs")
		assert.Contains(t, lines[3], "Validating config and starting app \"emit-metrics\"")
		assert.Contains(t, lines[4], "Started Dapr with app id \"emit-metrics\". HTTP Port: 3511.")
		assert.Contains(t, lines[5], "Writing log files to directory")
		assert.Contains(t, lines[5], "tests/apps/emit-metrics/.dapr/logs")
		assert.Contains(t, output, "Received signal to stop Dapr and app processes. Shutting down Dapr and app processes.")
		appTestOutput := AppTestOutput{
			appID:          "processor",
			baseLogDirPath: "../../apps/processor/.dapr/logs",
			appLogContents: []string{
				"Received metrics:  {1}",
			},
			daprdLogContent: []string{
				"HTTP server is running on port 3510",
				"You're up and running! Dapr logs will appear here.",
			},
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
		appTestOutput = AppTestOutput{
			appID:          "emit-metrics",
			baseLogDirPath: "../../apps/emit-metrics/.dapr/logs",
			appLogContents: []string{
				"DAPR_HTTP_PORT set to 3511",
				"DAPR_HOST_ADD set to localhost",
				"Metrics with ID 1 sent",
			},
			daprdLogContent: []string{
				"termination signal received: shutting down",
				"Exited Dapr successfully",
				"Exited App successfully",
			},
			daprdLogPollTimeout: 15 * time.Second,
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
	})

	t.Run("invalid template file env var not set", func(t *testing.T) {
		runFilePath := "../testdata/run-template-files/env_var_not_set_dapr.yaml"
		cmdUninstall()
		ensureDaprInstallation(t)

		t.Cleanup(func() {
			// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}
		outputCh := make(chan string)
		go func() {
			output, _ := cmdRun("", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		time.Sleep(time.Second * 10)
		cmdStopWithRunTemplate(runFilePath)
		var output string
		select {
		case output = <-outputCh:
		case <-time.After(25 * time.Second):
			t.Fatal("timed out waiting for run command to finish")
		}

		// Deterministic output for template file, so we can assert line by line
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 6, "expected at least 6 lines in output of starting two apps")
		assert.Contains(t, lines[1], "Started Dapr with app id \"processor\". HTTP Port: 3510.")
		assert.Contains(t, lines[2], "Writing log files to directory")
		assert.Contains(t, lines[2], "tests/apps/processor/.dapr/logs")
		assert.Contains(t, lines[4], "Started Dapr with app id \"emit-metrics\". HTTP Port: 3511.")
		assert.Contains(t, lines[5], "Writing log files to directory")
		assert.Contains(t, lines[5], "tests/apps/emit-metrics/.dapr/logs")
		assert.Contains(t, output, "Received signal to stop Dapr and app processes. Shutting down Dapr and app processes.")
		appTestOutput := AppTestOutput{
			appID:          "processor",
			baseLogDirPath: "../../apps/processor/.dapr/logs",
			daprdLogContent: []string{
				"HTTP server is running on port 3510",
				"You're up and running! Dapr logs will appear here.",
			},
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
		appTestOutput = AppTestOutput{
			appID:          "emit-metrics",
			baseLogDirPath: "../../apps/emit-metrics/.dapr/logs",
			appLogContents: []string{
				"DAPR_HTTP_PORT set to 3511",
				"exit status 1",
				"Error exiting App: exit status 1",
			},
			daprdLogContent: []string{
				"termination signal received: shutting down",
				"Exited Dapr successfully",
			},
			daprdLogPollTimeout: 15 * time.Second,
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
	})

	t.Run("valid template file no app command", func(t *testing.T) {
		cmdUninstall()
		ensureDaprInstallation(t)

		runFilePath := "../testdata/run-template-files/no_app_command.yaml"
		t.Cleanup(func() {
			// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}
		outputCh := make(chan string)
		go func() {
			output, _ := cmdRun("", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		time.Sleep(time.Second * 10)
		cmdStopWithRunTemplate(runFilePath)
		var output string
		select {
		case output = <-outputCh:
		case <-time.After(25 * time.Second):
			t.Fatal("timed out waiting for run command to finish")
		}

		// Deterministic output for template file, so we can assert line by line
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 7, "expected at least 7 lines in output of starting two apps with one app not having a command")
		assert.Contains(t, lines[1], "Started Dapr with app id \"processor\". HTTP Port: 3510.")
		assert.Contains(t, lines[2], "Writing log files to directory")
		assert.Contains(t, lines[2], "tests/apps/processor/.dapr/logs")
		assert.Contains(t, lines[4], "No application command found for app \"emit-metrics\" present in")
		assert.Contains(t, lines[5], "Started Dapr with app id \"emit-metrics\". HTTP Port: 3511.")
		assert.Contains(t, lines[6], "Writing log files to directory")
		assert.Contains(t, lines[6], "tests/apps/emit-metrics/.dapr/logs")
		assert.Contains(t, output, "Received signal to stop Dapr and app processes. Shutting down Dapr and app processes.")
		appTestOutput := AppTestOutput{
			appID:          "processor",
			baseLogDirPath: "../../apps/processor/.dapr/logs",
			appLogContents: []string{
				"Starting server in port 9086...",
				"termination signal received: shutting down",
			},
			daprdLogContent: []string{
				"HTTP server is running on port 3510",
				"You're up and running! Dapr logs will appear here.",
			},
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
		appTestOutput = AppTestOutput{
			appID:              "emit-metrics",
			baseLogDirPath:     "../../apps/emit-metrics/.dapr/logs",
			appLogDoesNotExist: true,
			daprdLogContent: []string{
				"termination signal received: shutting down",
				"Exited Dapr successfully",
			},
			daprdLogPollTimeout: 20 * time.Second,
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
	})

	t.Run("valid template file empty app command", func(t *testing.T) {
		cmdUninstall()
		ensureDaprInstallation(t)

		runFilePath := "../testdata/run-template-files/empty_app_command.yaml"
		t.Cleanup(func() {
			// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}
		outputCh := make(chan string)
		go func() {
			output, _ := cmdRun("", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		time.Sleep(time.Second * 10)
		cmdStopWithRunTemplate(runFilePath)
		var output string
		select {
		case output = <-outputCh:
		case <-time.After(25 * time.Second):
			t.Fatal("timed out waiting for run command to finish")
		}

		// Deterministic output for template file, so we can assert line by line
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 5, "expected at least 5 lines in output of starting two apps with last app having an empty command")
		assert.Contains(t, lines[1], "Started Dapr with app id \"processor\". HTTP Port: 3510.")
		assert.Contains(t, lines[2], "Writing log files to directory")
		assert.Contains(t, lines[2], "tests/apps/processor/.dapr/logs")
		assert.Contains(t, lines[4], "Error starting Dapr and app (\"emit-metrics\"): exec: no command")
		appTestOutput := AppTestOutput{
			appID:          "processor",
			baseLogDirPath: "../../apps/processor/.dapr/logs",
			appLogContents: []string{
				"Starting server in port 9084...",
				"termination signal received: shutting down",
			},
			daprdLogContent: []string{
				"HTTP server is running on port 3510",
				"You're up and running! Dapr logs will appear here.",
			},
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
		appTestOutput = AppTestOutput{
			appID:          "emit-metrics",
			baseLogDirPath: "../../apps/emit-metrics/.dapr/logs",
			appLogContents: []string{
				"Error starting app process: exec: no command",
			},
			daprdLogContent: []string{
				"Error starting Dapr and app (\"emit-metrics\"): exec: no command",
			},
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
	})

	t.Run("valid template file with app/daprd log destinations", func(t *testing.T) {
		cmdUninstall()
		ensureDaprInstallation(t)

		runFilePath := "../testdata/run-template-files/app_output_to_file_and_console.yaml"
		t.Cleanup(func() {
			// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}
		outputCh := make(chan string)
		go func() {
			output, _ := cmdRun("", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		time.Sleep(time.Second * 10)
		cmdStopWithRunTemplate(runFilePath)
		var output string
		select {
		case output = <-outputCh:
		case <-time.After(25 * time.Second):
			t.Fatal("timed out waiting for run command to finish")
		}

		// App logs for processor app should not be printed to console and only written to file.
		assert.NotContains(t, output, "== APP - processor")

		// Daprd logs for processor app should only be printed to console and not written to file.
		assert.Contains(t, output, "msg=\"All outstanding components processed\" app_id=processor")

		// App logs for emit-metrics app should be printed to console and written to file.
		assert.Contains(t, output, "== APP - emit-metrics")

		// Daprd logs for emit-metrics app should only be written to file.
		assert.NotContains(t, output, "msg=\"All outstanding components processed\" app_id=emit-metrics")

		assert.Contains(t, output, "Received signal to stop Dapr and app processes. Shutting down Dapr and app processes.")

		appTestOutput := AppTestOutput{
			appID:          "processor",
			baseLogDirPath: "../../apps/processor/.dapr/logs",
			appLogContents: []string{
				"Received metrics:  {1}",
			},
			daprdLogContent:          []string{},
			daprdLogFileDoesNotExist: true,
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
		appTestOutput = AppTestOutput{
			appID:          "emit-metrics",
			baseLogDirPath: "../../apps/emit-metrics/.dapr/logs",
			appLogContents: []string{
				"DAPR_HTTP_PORT set to 3511",
				"DAPR_HOST_ADD set to localhost",
				"Metrics with ID 1 sent",
			},
			daprdLogContent: []string{
				"termination signal received: shutting down",
				"Exited Dapr successfully",
				"Exited App successfully",
			},
			daprdLogPollTimeout: 20 * time.Second,
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
	})
}

func TestRunTemplateFileWithoutDaprInit(t *testing.T) {
	// remove any dapr installation before this test.
	must(t, cmdUninstall, "failed to uninstall Dapr")
	t.Run("valid template file without dapr init", func(t *testing.T) {
		t.Cleanup(func() {
			// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
			cleanUpLogs()
		})
		args := []string{
			"-f", "../testdata/run-template-files/no_app_command.yaml",
		}
		output, err := cmdRun("", args...)
		t.Logf("%s", output)
		require.Error(t, err, "run must fail")
		assert.Contains(t, output, "Error starting Dapr and app (\"processor\"): fork/exec")
		assert.Contains(t, output, "daprd: no such file or directory")
	})
}

func assertLogOutputForRunTemplateExec(t *testing.T, appTestOutput AppTestOutput) {
	// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
	// This is true for the tests in this file.
	if !appTestOutput.daprdLogFileDoesNotExist {
		daprdLogFileName, err := lookUpFileFullName(appTestOutput.baseLogDirPath, "daprd")
		require.NoError(t, err, "failed to find daprd log file")
		daprdLogPath := filepath.Join(appTestOutput.baseLogDirPath, daprdLogFileName)
		readAndAssertLogFileContents(t, daprdLogPath, appTestOutput.daprdLogContent, appTestOutput.daprdLogPollTimeout)
	}
	if appTestOutput.appLogDoesNotExist {
		return
	}
	appLogFileName, err := lookUpFileFullName(appTestOutput.baseLogDirPath, "app")
	require.NoError(t, err, "failed to find app log file")
	appLogPath := filepath.Join(appTestOutput.baseLogDirPath, appLogFileName)
	readAndAssertLogFileContents(t, appLogPath, appTestOutput.appLogContents, 0)
}

func readAndAssertLogFileContents(t *testing.T, logFilePath string, expectedContent []string, pollTimeout time.Duration) {
	assert.FileExists(t, logFilePath, "log file %s must exist", logFilePath)
	if pollTimeout > 0 {
		require.Eventually(t, func() bool {
			fileContents, err := ioutil.ReadFile(logFilePath)
			if err != nil {
				return false
			}
			contentString := string(fileContents)
			for _, line := range expectedContent {
				if !strings.Contains(contentString, line) {
					return false
				}
			}
			return true
		}, pollTimeout, 300*time.Millisecond,
			"log file %s did not contain all expected lines within %v (expected: %v)", logFilePath, pollTimeout, expectedContent)
		return
	}
	fileContents, err := ioutil.ReadFile(logFilePath)
	require.NoError(t, err, "failed to read %s log", logFilePath)
	contentString := string(fileContents)
	for _, line := range expectedContent {
		assert.Containsf(t, contentString, line, "expected logline to be present, line=%s", line)
	}
}

// lookUpFileFullName looks up the full name of the first file with partial name match in the directory.
func lookUpFileFullName(dirPath, partialFilename string) (string, error) {
	// Look for the file in the current directory
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return "", err
	}
	for _, file := range files {
		if strings.Contains(file.Name(), partialFilename) {
			return file.Name(), nil
		}
	}
	return "", fmt.Errorf("failed to find file with partial name %s in directory %s", partialFilename, dirPath)
}
