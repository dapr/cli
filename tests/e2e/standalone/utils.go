//go:build e2e || template

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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapr/cli/tests/e2e/common"
)

// getSocketCases return different unix socket paths for testing across Dapr commands.
// If the tests are being run on Windows, it returns an empty array.
func getSocketCases() []string {
	if runtime.GOOS == "windows" {
		return []string{""}
	} else {
		return []string{"", "/tmp"}
	}
}

// must is a helper function that executes a function and expects it to succeed.
func must(t *testing.T, f func(args ...string) (string, error), message string, fArgs ...string) {
	_, err := f(fArgs...)
	require.NoError(t, err, message)
}

// checkAndWriteFile writes content to file if it does not exist.
func checkAndWriteFile(filePath string, b []byte) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// #nosec G306
		if err = os.WriteFile(filePath, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// isSlimMode returns true if DAPR_E2E_INIT_SLIM is set to true.
func isSlimMode() bool {
	return os.Getenv("DAPR_E2E_INIT_SLIM") == "true"
}

// createSlimComponents creates default state store and pubsub components in path.
func createSlimComponents(path string) error {
	components := map[string]string{
		"pubsub.yaml": `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
    name: pubsub
spec:
    type: pubsub.in-memory
    version: v1
    metadata: []`,
		"statestore.yaml": `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
    name: statestore
spec:
    type: state.in-memory
    version: v1
    metadata: []`,
	}

	for fileName, content := range components {
		fullPath := filepath.Join(path, fileName)
		if err := checkAndWriteFile(fullPath, []byte(content)); err != nil {
			return err
		}
	}

	return nil
}

// executeAgainstRunningDapr runs a function against a running Dapr instance.
// If Dapr or the App throws an error, the test is marked as failed.
// After f() returns the process is given 60s to exit on its own (f()
// should have called `dapr stop`). If it hasn't exited by then, the
// process is killed so the test doesn't hang until the global 40m timeout.
func executeAgainstRunningDapr(t *testing.T, f func(), daprArgs ...string) {
	daprPath := common.GetDaprPath()

	cmd := exec.Command(daprPath, daprArgs...)
	reader, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(reader)

	cmd.Start()

	// scanDone is closed when the scanner.Scan loop finishes, meaning
	// the process has closed its stdout pipe (i.e., is exiting).
	scanDone := make(chan struct{})

	// Safety goroutine: kill the process if it is still running after
	// 5 minutes. This prevents a 40-minute hang when f() blocks
	// (e.g. a subtest hangs on a channel receive) or when f() fails
	// to stop daprd. Killing the process closes the stdout pipe,
	// which unblocks scanner.Scan() below.
	go func() {
		select {
		case <-time.After(5 * time.Minute):
			t.Log("executeAgainstRunningDapr: process did not exit within 5m, killing")
			cmd.Process.Kill()
		case <-scanDone:
			// Process exited on its own — nothing to do.
		}
	}()

	daprOutput := ""
	for scanner.Scan() {
		outputChunk := scanner.Text()
		t.Log(outputChunk)
		if strings.Contains(outputChunk, "You're up and running!") {
			f()
		}
		daprOutput += outputChunk
	}
	close(scanDone)

	err := cmd.Wait()
	hasAppCommand := !strings.Contains(daprOutput, "WARNING: no application command found")
	terminatedBySignal := strings.Contains(daprOutput, "terminated signal received: shutting down")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 &&
			strings.Contains(daprOutput, "Exited Dapr successfully") &&
			(!hasAppCommand || terminatedBySignal || strings.Contains(daprOutput, "Exited App successfully")) {
			err = nil
		}
	}
	require.NoError(t, err, "dapr didn't exit cleanly")
	assert.NotContains(t, daprOutput, "The App process exited with error code: exit status", "Stop command should have been called before the app had a chance to exit")
	assert.Contains(t, daprOutput, "Exited Dapr successfully")
	if hasAppCommand && !terminatedBySignal {
		assert.Contains(t, daprOutput, "Exited App successfully")
	}
}

// waitForPortsFree polls until all given ports are available for binding.
// This prevents port contention between sequential tests that reuse
// hardcoded ports (e.g. container ports from dapr init).
func waitForPortsFree(t *testing.T, ports ...int) {
	t.Helper()
	require.Eventually(t, func() bool {
		for _, port := range ports {
			ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				return false
			}
			ln.Close()
		}
		return true
	}, 60*time.Second, time.Second, "ports %v not available in time", ports)
}

// startDaprRun starts `dapr run` in a background goroutine and registers
// cleanup handlers that stop the app and wait for the goroutine to finish.
// This prevents "Log in goroutine after Test has completed" panics that
// occur when the cmdRun goroutine outlives the test.
//
// stopArgs is passed to cmdStopWithAppID or cmdStopWithRunTemplate depending
// on whether it looks like a file path (contains "/" or ".yaml").
func startDaprRun(t *testing.T, ports []int, stopFn func(), runArgs ...string) {
	t.Helper()

	if len(ports) > 0 {
		waitForPortsFree(t, ports...)
	}

	var wg sync.WaitGroup
	// Register wg.Wait first so it runs last (LIFO cleanup order).
	t.Cleanup(func() { wg.Wait() })
	t.Cleanup(stopFn)

	wg.Add(1)
	go func() {
		defer wg.Done()
		o, _ := cmdRun("", runArgs...)
		// Only safe to call t.Log here because cleanup waits for us
		// via wg.Wait().
		t.Log(o)
	}()
}

// startDaprRunRetry is like startDaprRun but retries cmdRun up to 10 times
// on failure. Used by scheduler tests where port contention can cause
// transient startup failures.
func startDaprRunRetry(t *testing.T, ports []int, stopFn func(), runArgs ...string) {
	t.Helper()

	if len(ports) > 0 {
		waitForPortsFree(t, ports...)
	}

	var wg sync.WaitGroup
	t.Cleanup(func() { wg.Wait() })
	t.Cleanup(stopFn)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10 {
			o, err := cmdRun("", runArgs...)
			t.Log(o)
			if err == nil {
				break
			}
			t.Log(err)
			time.Sleep(time.Second * 2)
		}
	}()
}

// ensureDaprInstallation ensures that Dapr is installed.
// If Dapr is not installed, a new installation is attempted.
func ensureDaprInstallation(t *testing.T) {
	daprRuntimeVersion, daprDashboardVersion := common.GetVersionsFromEnv(t, false)
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "failed to get user home directory")

	daprPath := filepath.Join(homeDir, ".dapr")
	_, err = os.Stat(daprPath)
	if os.IsNotExist(err) {
		// Wait for container ports from a previous dapr installation to
		// be fully released. On macOS, container port bindings can linger
		// briefly after `dapr uninstall` removes the containers.
		if !isSlimMode() {
			waitForPortsFree(t,
				58080, // placement health
				58081, // scheduler health
				50005, // placement gRPC
			)
		}
		args := []string{
			"--runtime-version", daprRuntimeVersion,
			"--dashboard-version", daprDashboardVersion,
		}
		output, err := cmdInit(args...)
		require.NoError(t, err, "failed to install dapr:%v", output)
	} else if err != nil {
		// Some other error occurred.
		require.NoError(t, err, "failed to stat dapr installation")
	}

	// Slim mode does not have any components by default.
	// Install the components required by the tests.
	if isSlimMode() {
		err = createSlimComponents(filepath.Join(daprPath, "components"))
		require.NoError(t, err, "failed to create components")
	}
}

func containerRuntime() string {
	if daprContainerRuntime, ok := os.LookupEnv("CONTAINER_RUNTIME"); ok {
		return daprContainerRuntime
	}
	return ""
}

func getRunningProcesses() []string {
	cmd := exec.Command("ps", "-o", "pid,command")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	processes := strings.Split(string(output), "\n")

	// clean the process output whitespace
	for i, process := range processes {
		processes[i] = strings.TrimSpace(process)
	}
	return processes
}

func stopProcess(args ...string) error {
	processCommand := strings.Join(args, " ")
	processes := getRunningProcesses()
	for _, process := range processes {
		if strings.Contains(process, processCommand) {
			processSplit := strings.SplitN(process, " ", 2)
			cmd := exec.Command("kill", "-9", processSplit[0])
			err := cmd.Run()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func cleanUpLogs() {
	os.RemoveAll("../../apps/emit-metrics/.dapr/logs")
	os.RemoveAll("../../apps/processor/.dapr/logs")
}

// lookUpFileFullName looks up the full name of the first file with partial name match in the directory.
func lookUpFileFullName(dirPath, partialFilename string) (string, error) {
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
