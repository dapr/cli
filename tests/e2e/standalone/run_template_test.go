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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var httpClient = &http.Client{Timeout: 500 * time.Millisecond}

// waitForDaprHealth polls the Dapr HTTP healthz endpoints until all
// sidecars report healthy. This confirms both the sidecar and its app
// are running, independent of log output timing.
func waitForDaprHealth(t *testing.T, timeout time.Duration, httpPorts ...int) {
	t.Helper()
	require.Eventually(t, func() bool {
		for _, port := range httpPorts {
			resp, err := httpClient.Get(fmt.Sprintf("http://localhost:%d/v1.0/healthz", port))
			if err != nil {
				return false
			}
			resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return false
			}
		}
		return true
	}, timeout, 500*time.Millisecond, "dapr sidecars on ports %v not healthy within %v", httpPorts, timeout)
}

// waitForAppHealthy polls dapr list to discover the HTTP port for the
// given appID, then health-checks it. Use this when the HTTP port is
// auto-assigned and not known in advance.
func waitForAppHealthy(t *testing.T, timeout time.Duration, appID string) {
	t.Helper()
	require.Eventually(t, func() bool {
		output, err := cmdList("json")
		if err != nil {
			return false
		}
		var result []map[string]interface{}
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			return false
		}
		for _, entry := range result {
			if entry["appId"] != appID {
				continue
			}
			httpPort, _ := entry["httpPort"].(float64)
			if httpPort <= 0 {
				return false
			}
			resp, err := httpClient.Get(fmt.Sprintf("http://localhost:%d/v1.0/healthz", int(httpPort)))
			if err != nil {
				return false
			}
			resp.Body.Close()
			return resp.StatusCode >= 200 && resp.StatusCode < 300
		}
		return false
	}, timeout, time.Second, "dapr app %q not healthy within %v", appID, timeout)
}

// waitForAppsListed polls dapr list until all given appIDs are present with
// a non-zero HTTP port. Unlike waitForDaprHealth this does NOT check the
// healthz endpoint, so it works in slim mode where placement/scheduler are
// absent. It guarantees that daprd is up, listening, and has stored metadata —
// which is the prerequisite for `dapr stop -f` to locate the CLI process.
func waitForAppsListed(t *testing.T, timeout time.Duration, appIDs ...string) {
	t.Helper()
	require.Eventually(t, func() bool {
		output, err := cmdList("json")
		if err != nil {
			return false
		}
		var result []map[string]interface{}
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			return false
		}
		found := 0
		for _, id := range appIDs {
			for _, entry := range result {
				if entry["appId"] == id {
					httpPort, _ := entry["httpPort"].(float64)
					if httpPort > 0 {
						found++
						break
					}
				}
			}
		}
		return found == len(appIDs)
	}, timeout, time.Second, "dapr apps %v not listed within %v", appIDs, timeout)
}

// waitForLogContent polls until the log file matching partialFileName in
// dirPath contains the expected substring. This is used to wait for slow
// app startup (e.g. `go run` compilation) before proceeding with the test.
func waitForLogContent(t *testing.T, dirPath, partialFileName, expected string, timeout time.Duration) {
	t.Helper()
	require.Eventually(t, func() bool {
		fileName, err := lookUpFileFullName(dirPath, partialFileName)
		if err != nil {
			return false
		}
		contents, err := ioutil.ReadFile(filepath.Join(dirPath, fileName))
		if err != nil {
			return false
		}
		return strings.Contains(string(contents), expected)
	}, timeout, time.Second, "log file matching %q in %s did not contain %q within %v", partialFileName, dirPath, expected, timeout)
}

// collectOutput waits for the CLI process output from outputCh. If the
// output does not arrive within timeout, the context is canceled (which
// SIGKILL's the CLI via exec.CommandContext) and we wait a further 20s
// for WaitDelay to close pipes and CombinedOutput to return.
func collectOutput(t *testing.T, outputCh <-chan string, cancel context.CancelFunc, timeout time.Duration) string {
	t.Helper()
	select {
	case output := <-outputCh:
		return output
	case <-time.After(timeout):
		cancel()
		select {
		case output := <-outputCh:
			return output
		case <-time.After(20 * time.Second):
			t.Fatal("timed out waiting for run command to finish")
			return ""
		}
	}
}

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
