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
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
func executeAgainstRunningDapr(t *testing.T, f func(), daprArgs ...string) {
	daprPath := common.GetDaprPath()

	cmd := exec.Command(daprPath, daprArgs...)
	reader, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(reader)

	cmd.Start()

	daprOutput := ""
	for scanner.Scan() {
		outputChunk := scanner.Text()
		t.Log(outputChunk)
		if strings.Contains(outputChunk, "You're up and running!") {
			f()
		}
		daprOutput += outputChunk
	}

	err := cmd.Wait()
	hasAppCommand := !strings.Contains(daprOutput, "WARNING: no application command found")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 &&
			strings.Contains(daprOutput, "Exited Dapr successfully") &&
			(!hasAppCommand || strings.Contains(daprOutput, "Exited App successfully")) {
			err = nil
		}
	}
	require.NoError(t, err, "dapr didn't exit cleanly")
	assert.NotContains(t, daprOutput, "The App process exited with error code: exit status", "Stop command should have been called before the app had a chance to exit")
	assert.Contains(t, daprOutput, "Exited Dapr successfully")
	if hasAppCommand {
		assert.Contains(t, daprOutput, "Exited App successfully")
	}
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
