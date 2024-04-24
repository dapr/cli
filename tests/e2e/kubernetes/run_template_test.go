//go:build e2e || templatek8s

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

package kubernetes_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/dapr/cli/tests/e2e/common"
	"github.com/dapr/cli/tests/e2e/spawn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	waitForRunOutput   = 60 * time.Second
	windowsOsType      = "windows"
	serviceYamlFile    = "service.yaml"
	deploymentYamlFile = "deployment.yaml"
)

var (
	nodeAppBaseDaprDir   = filepath.Join("..", "..", "apps", "nodeapp", ".dapr")
	pythonAppBaseDaprDir = filepath.Join("..", "..", "apps", "pythonapp", ".dapr")
	nodeAppLogsDir       = filepath.Join(nodeAppBaseDaprDir, "logs")
	pythonAppLogsDir     = filepath.Join(pythonAppBaseDaprDir, "logs")
	nodeAppDeployDir     = filepath.Join(nodeAppBaseDaprDir, "deploy")
	pythonappDeployDir   = filepath.Join(pythonAppBaseDaprDir, "deploy")
)

func TestKubernetesRunFile(t *testing.T) {
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	opts := common.TestOptions{
		DevEnabled:  true,
		HAEnabled:   false,
		MTLSEnabled: true,
	}
	tests = append(tests, common.GetInstallOnlyTest(currentVersionDetails, opts))

	tests = append(tests, common.TestCase{
		Name:     "run file k8s",
		Callable: testRunFile(common.TestOptions{}),
	})

	opts = common.TestOptions{
		DevEnabled:   true,
		UninstallAll: true,
	}

	tests = append(tests, common.GetUninstallOnlyTest(currentVersionDetails, opts))

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func testRunFile(opts common.TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		// File present as part of "tests/e2e/testdata" folder.
		runFilePath := filepath.Join("..", "testdata", "run-template-files", "dapr-k8s.yaml")
		t.Cleanup(func() {
			// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
			os.RemoveAll(nodeAppLogsDir)
			os.RemoveAll(pythonAppLogsDir)
			stopAllApps(t, runFilePath)
		})
		go startAppsWithTemplateFile(t, runFilePath)
		time.Sleep(waitForRunOutput)

		// assert yaml files created.
		assert.FileExists(t, filepath.Join(nodeAppDeployDir, serviceYamlFile), "service yaml must exist for node app")
		assert.FileExists(t, filepath.Join(nodeAppDeployDir, deploymentYamlFile), "deployment yaml must exist for node app")

		assert.FileExists(t, filepath.Join(pythonappDeployDir, deploymentYamlFile), "deployment yaml must exist for python app")

		_, err := cmdStopWithRunTemplate(runFilePath)
		assert.NoError(t, err, "failed to stop apps started with run template")
		time.Sleep(5 * time.Second)

		// For Node app
		daprdLogFile, err := lookUpFileFullName(nodeAppLogsDir, "daprd")
		require.NoError(t, err, "expected no error in finding the daprd log file for node app")
		contents := []string{
			"dapr initialized. Status: Running.",
			"app_id=nodeapp",
			"Shutting down component statestore (state.redis/v1)",
			"Shutting down component pubsub (pubsub.redis/v1)",
		}
		assertLogFileContains(t, filepath.Join(nodeAppLogsDir, daprdLogFile), contents)

		appLogFile, err := lookUpFileFullName(nodeAppLogsDir, "app")
		require.NoError(t, err, "expected no error in finding the app log file for node app")
		contents = []string{
			"== APP - nodeapp == Node App listening on port 3000!",
			// not specifying any order ID as it is non-deterministic and dependent on network and OS.
			"== APP - nodeapp == Got a new order! Order ID:",
			"== APP - nodeapp == Successfully persisted state for Order ID:",
		}
		assertLogFileContains(t, filepath.Join(nodeAppLogsDir, appLogFile), contents)

		// For Python app

		daprdLogFile, err = lookUpFileFullName(pythonAppLogsDir, "daprd")
		require.NoError(t, err, "expected no error in finding the daprd log file for python app")
		contents = []string{
			"dapr initialized. Status: Running.",
			"app_id=pythonapp",
			"Shutting down component statestore (state.redis/v1)",
			"Shutting down component pubsub (pubsub.redis/v1)",
		}
		assertLogFileContains(t, filepath.Join(pythonAppLogsDir, daprdLogFile), contents)

		appLogFile, err = lookUpFileFullName(pythonAppLogsDir, "app")
		require.NoError(t, err, "expected no error in finding the app log file for python app")
		contents = []string{
			// logs during shutdown sequence.
			"== APP - pythonapp == HTTP 500 => {\"errorCode\":\"ERR_DIRECT_INVOKE\",\"message\":\"failed to invoke, id: nodeapp",
		}
		assertLogFileContains(t, filepath.Join(pythonAppLogsDir, appLogFile), contents)
	}
}

func startAppsWithTemplateFile(t *testing.T, runFilePath string) {
	// All apps are withing "tests/apps" folder.
	args := []string{
		"-f", runFilePath,
		"-k",
	}
	output, err := cmdRun(args...)
	t.Logf(output)
	require.NoError(t, err, "run failed")
	assert.Contains(t, output, "Deploying service YAML")
	assert.Contains(t, output, "Deploying deployment YAML")

	assert.Contains(t, output, "This is a preview feature and subject to change in future releases.")
	assert.Contains(t, output, "Validating config and starting app \"nodeapp\"")
	assert.Contains(t, output, "Deploying app \"nodeapp\" to Kubernetes")
	if runtime.GOOS == windowsOsType {
		assert.Contains(t, output, "tests\\apps\\nodeapp\\.dapr\\deploy\\service.yaml\" to Kubernetes")
	} else {
		assert.Contains(t, output, "tests/apps/nodeapp/.dapr/deploy/service.yaml")
	}

	if runtime.GOOS == windowsOsType {
		assert.Contains(t, output, "tests\\apps\\nodeapp\\.dapr\\deploy\\deployment.yaml\" to Kubernetes")
	} else {
		assert.Contains(t, output, "tests/apps/nodeapp/.dapr/deploy/deployment.yaml\" to Kubernetes")
	}
	assert.Contains(t, output, "Streaming logs for containers in pod \"nodeapp-")
	if runtime.GOOS == windowsOsType {
		assert.Contains(t, output, "tests\\apps\\nodeapp\\.dapr\\logs")
	} else {
		assert.Contains(t, output, "tests/apps/nodeapp/.dapr/logs")
	}
	assert.Contains(t, output, "Validating config and starting app \"pythonapp\"")
	if runtime.GOOS == windowsOsType {
		assert.Contains(t, output, "tests\\apps\\pythonapp\\.dapr\\deploy\\deployment.yaml\" to Kubernetes")
	} else {
		assert.Contains(t, output, "tests/apps/pythonapp/.dapr/deploy/deployment.yaml\" to Kubernetes")
	}
	assert.Contains(t, output, "Streaming logs for containers in pod \"pythonapp-")
	if runtime.GOOS == windowsOsType {
		assert.Contains(t, output, "tests\\apps\\pythonapp\\.dapr\\logs")
	} else {
		assert.Contains(t, output, "tests/apps/pythonapp/.dapr/logs")
	}
	assert.Contains(t, output, "Starting to monitor Kubernetes pods for deletion.")
}

// cmdRun runs a Dapr instance and returns the command output and error.
func cmdRun(args ...string) (string, error) {
	runArgs := []string{"run"}

	runArgs = append(runArgs, args...)
	return spawn.Command(common.GetDaprPath(), runArgs...)
}

// cmdStopWithRunTemplate stops the apps started with run template file and returns the command output and error.
func cmdStopWithRunTemplate(runTemplateFile string, args ...string) (string, error) {
	stopArgs := append([]string{"stop", "--log-as-json", "-k", "-f", runTemplateFile}, args...)
	return spawn.Command(common.GetDaprPath(), stopArgs...)
}

func assertLogFileContains(t *testing.T, logFilePath string, expectedContent []string) {
	assert.FileExists(t, logFilePath, "log file %s must exist", logFilePath)
	fileContents, err := os.ReadFile(logFilePath)
	assert.NoError(t, err, "failed to read %s log", logFilePath)
	contentString := string(fileContents)
	for _, line := range expectedContent {
		assert.Contains(t, contentString, line, "expected logline to be present")
	}
}

// lookUpFileFullName looks up the full name of the first file with partial name match in the directory.
func lookUpFileFullName(dirPath, partialFilename string) (string, error) {
	// Look for the file in the current directory
	files, err := os.ReadDir(dirPath)
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

func stopAllApps(t *testing.T, runfile string) {
	_, err := cmdStopWithRunTemplate(runfile)
	require.NoError(t, err, "failed to stop apps")
	time.Sleep(5 * time.Second)
}
