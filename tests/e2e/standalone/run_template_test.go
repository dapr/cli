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
	"context"
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
	cleanUpLogs()
	// These tests are dependent on run template files in ../testdata/run-template-files folder.

	t.Run("invalid template file wrong emit metrics app run", func(t *testing.T) {
		runFilePath := "../testdata/run-template-files/wrong_emit_metrics_app_dapr.yaml"
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()
		t.Cleanup(func() {
			cmdStopWithRunTemplate(runFilePath)
			cmdStopWithAppID("processor")
			cmdStopWithAppID("emit-metrics")
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}

		waitForPortsFree(t, 3510, 3511)
		outputCh := make(chan string, 1)
		go func() {
			output, _ := cmdRunWithContext(ctx, "", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		// Wait for the emit-metrics app to fail (wrong file name). The app
		// log gets written quickly since `go run wrongappname.go` fails
		// immediately. Then send stop so the CLI shuts down gracefully.
		waitForLogContent(t, "../../apps/emit-metrics/.dapr/logs", "app", "exit status 1", 60*time.Second)
		cmdStopWithRunTemplate(runFilePath)
		// Give the CLI time to gracefully shut down. The CLI must process
		// the SIGTERM from stop, then kill daprd/app processes (up to 5s
		// grace period each). 60s is generous.
		output := collectOutput(t, outputCh, cancel, 60*time.Second)

		assert.Contains(t, output, "Started Dapr with app id \"processor\". HTTP Port: 3510.")
		assert.Contains(t, output, "Writing log files to directory")
		assert.Contains(t, output, "tests/apps/processor/.dapr/logs")
		assert.Contains(t, output, "Started Dapr with app id \"emit-metrics\". HTTP Port: 3511.")
		assert.Contains(t, output, "tests/apps/emit-metrics/.dapr/logs")
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
		if isSlimMode() {
			t.Skip("skipping: slim mode has no placement/scheduler so daprd cannot become healthy")
		}

		runFilePath := "../testdata/run-template-files/dapr.yaml"
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()
		t.Cleanup(func() {
			cmdStopWithRunTemplate(runFilePath)
			cmdStopWithAppID("processor")
			cmdStopWithAppID("emit-metrics")
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}

		waitForPortsFree(t, 3510, 3511)
		outputCh := make(chan string, 1)
		go func() {
			output, _ := cmdRunWithContext(ctx, "", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		waitForDaprHealth(t, 60*time.Second, 3510, 3511)
		waitForLogContent(t, "../../apps/emit-metrics/.dapr/logs", "app", "Metrics with ID 1 sent", 60*time.Second)
		cmdStopWithRunTemplate(runFilePath)
		output := collectOutput(t, outputCh, cancel, 60*time.Second)

		// Deterministic output for template file, so we can assert line by line
		lines := strings.Split(output, "\n")
		require.GreaterOrEqual(t, len(lines), 6, "expected at least 6 lines in output of starting two apps")
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

		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()
		t.Cleanup(func() {
			cmdStopWithRunTemplate(runFilePath)
			cmdStopWithAppID("processor")
			cmdStopWithAppID("emit-metrics")
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}
		waitForPortsFree(t, 3510, 3511)
		outputCh := make(chan string, 1)
		go func() {
			output, _ := cmdRunWithContext(ctx, "", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		// The emit-metrics app must compile (go run) and then fail because
		// the env var is not set. This can be slow on CI (downloading deps,
		// compiling). Wait for the app log to confirm the app has failed
		// before sending stop — otherwise stop kills the app before it can
		// produce the expected error output.
		waitForLogContent(t, "../../apps/emit-metrics/.dapr/logs", "app", "exit status 1", 90*time.Second)
		cmdStopWithRunTemplate(runFilePath)
		output := collectOutput(t, outputCh, cancel, 60*time.Second)

		assert.Contains(t, output, "Started Dapr with app id \"processor\". HTTP Port: 3510.")
		assert.Contains(t, output, "Writing log files to directory")
		assert.Contains(t, output, "tests/apps/processor/.dapr/logs")
		assert.Contains(t, output, "Started Dapr with app id \"emit-metrics\". HTTP Port: 3511.")
		assert.Contains(t, output, "tests/apps/emit-metrics/.dapr/logs")
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
		if isSlimMode() {
			t.Skip("skipping: slim mode has no placement/scheduler so daprd cannot become healthy")
		}

		runFilePath := "../testdata/run-template-files/no_app_command.yaml"
		// The CLI performs daprd health checks (IsDaprListeningOnPort) for
		// apps with appPort=0. Each check can take up to 60s. With two
		// ports (HTTP + gRPC) per app, the total startup can take >120s.
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()
		t.Cleanup(func() {
			cmdStopWithRunTemplate(runFilePath)
			cmdStopWithAppID("processor")
			cmdStopWithAppID("emit-metrics")
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}
		waitForPortsFree(t, 3510, 3511)
		outputCh := make(chan string, 1)
		go func() {
			output, _ := cmdRunWithContext(ctx, "", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		// Wait for emit-metrics to be fully healthy before stopping.
		// NOTE: Do NOT use waitForAppsListed here — it detects the
		// daprd process BEFORE the CLI finishes health checks, causing
		// a race where stop is sent too early. waitForAppHealthy also
		// checks the healthz endpoint, confirming the sidecar is ready.
		waitForAppHealthy(t, 180*time.Second, "emit-metrics")
		cmdStopWithRunTemplate(runFilePath)
		output := collectOutput(t, outputCh, cancel, 60*time.Second)

		assert.Contains(t, output, "Started Dapr with app id \"processor\". HTTP Port: 3510.")
		assert.Contains(t, output, "Writing log files to directory")
		assert.Contains(t, output, "tests/apps/processor/.dapr/logs")
		assert.Contains(t, output, "No application command found for app \"emit-metrics\" present in")
		assert.Contains(t, output, "Started Dapr with app id \"emit-metrics\". HTTP Port: 3511.")
		assert.Contains(t, output, "tests/apps/emit-metrics/.dapr/logs")
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
			daprdLogPollTimeout: 60 * time.Second,
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
	})

	t.Run("valid template file empty app command", func(t *testing.T) {
		if isSlimMode() {
			t.Skip("skipping: slim mode has no placement/scheduler so daprd cannot become healthy")
		}

		runFilePath := "../testdata/run-template-files/empty_app_command.yaml"
		// The CLI starts daprd for emit-metrics, runs health checks (up
		// to 60s each for HTTP and gRPC ports since appPort=0), detects
		// the empty command, kills daprd, and exits with error. The whole
		// process can take >120s on slow runners.
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()
		t.Cleanup(func() {
			cmdStopWithRunTemplate(runFilePath)
			cmdStopWithAppID("processor")
			cmdStopWithAppID("emit-metrics")
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}
		waitForPortsFree(t, 3510, 3511)
		outputCh := make(chan string, 1)
		go func() {
			output, _ := cmdRunWithContext(ctx, "", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		// The CLI exits on its own after detecting the empty command
		// (exitWithError=true). Do NOT send cmdStopWithRunTemplate here:
		// the SIGTERM would sit in sigCh unread while the CLI is blocked
		// in daprd health checks. Just wait for the CLI to finish.
		output := collectOutput(t, outputCh, cancel, 180*time.Second)

		assert.Contains(t, output, "Started Dapr with app id \"processor\". HTTP Port: 3510.")
		assert.Contains(t, output, "Writing log files to directory")
		assert.Contains(t, output, "tests/apps/processor/.dapr/logs")
		assert.Contains(t, output, "Error starting Dapr and app (\"emit-metrics\"): exec: no command")
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
		if isSlimMode() {
			t.Skip("skipping: slim mode has no placement/scheduler so daprd cannot become healthy")
		}

		runFilePath := "../testdata/run-template-files/app_output_to_file_and_console.yaml"
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()
		t.Cleanup(func() {
			cmdStopWithRunTemplate(runFilePath)
			cmdStopWithAppID("processor")
			cmdStopWithAppID("emit-metrics")
			cleanUpLogs()
		})
		args := []string{
			"-f", runFilePath,
		}
		waitForPortsFree(t, 3510, 3511)
		outputCh := make(chan string, 1)
		go func() {
			output, _ := cmdRunWithContext(ctx, "", args...)
			t.Logf("%s", output)
			outputCh <- output
		}()
		waitForDaprHealth(t, 60*time.Second, 3510, 3511)
		waitForLogContent(t, "../../apps/emit-metrics/.dapr/logs", "app", "Metrics with ID 1 sent", 60*time.Second)
		cmdStopWithRunTemplate(runFilePath)
		output := collectOutput(t, outputCh, cancel, 60*time.Second)

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
	// Remove dapr installation so we can test running without init.
	must(t, cmdUninstall, "failed to uninstall Dapr")
	// Reinstall Dapr when done so subsequent tests still work.
	t.Cleanup(func() {
		ensureDaprInstallation(t)
	})
	t.Run("valid template file without dapr init", func(t *testing.T) {
		t.Cleanup(func() {
			// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
			cleanUpLogs()
		})
		waitForPortsFree(t, 3510, 3511)
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



