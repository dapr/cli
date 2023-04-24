//go:build e2e || template
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
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopAppsStartedWithRunTemplate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		// remove dapr installation after all tests in this function.
		tearDownTestSetup(t)
	})

	t.Run("stop apps by passing run template file", func(t *testing.T) {
		go ensureAllAppsStartedWithRunTemplate(t)
		time.Sleep(10 * time.Second)
		cliPID := getCLIPID(t)
		output, err := cmdStopWithRunTemplate("../testdata/run-template-files/dapr.yaml")
		assert.NoError(t, err, "failed to stop apps started with run template")
		assert.Contains(t, output, "Dapr and app processes stopped successfully")
		verifyCLIPIDNotExist(t, cliPID)
	})

	t.Run("stop apps by passing a directory containing dapr.yaml", func(t *testing.T) {
		go ensureAllAppsStartedWithRunTemplate(t)
		time.Sleep(10 * time.Second)
		cliPID := getCLIPID(t)
		output, err := cmdStopWithRunTemplate("../testdata/run-template-files")
		assert.NoError(t, err, "failed to stop apps started with run template")
		assert.Contains(t, output, "Dapr and app processes stopped successfully")
		verifyCLIPIDNotExist(t, cliPID)
	})

	t.Run("stop apps by passing an invalid directory", func(t *testing.T) {
		go ensureAllAppsStartedWithRunTemplate(t)
		time.Sleep(10 * time.Second)
		output, err := cmdStopWithRunTemplate("../testdata/invalid-dir")
		assert.Contains(t, output, "Failed to get run file path")
		assert.Error(t, err, "failed to stop apps started with run template")
		// cleanup started apps
		output, err = cmdStopWithRunTemplate("../testdata/run-template-files")
		assert.NoError(t, err, "failed to stop apps started with run template")
		assert.Contains(t, output, "Dapr and app processes stopped successfully")
	})

	t.Run("stop apps started with run template", func(t *testing.T) {
		go ensureAllAppsStartedWithRunTemplate(t)
		time.Sleep(10 * time.Second)
		cliPID := getCLIPID(t)
		output, err := cmdStopWithAppID("emit-metrics", "processor")
		assert.NoError(t, err, "failed to stop apps started with run template")
		assert.Contains(t, output, "app stopped successfully: emit-metrics")
		assert.Contains(t, output, "app stopped successfully: processor")
		assert.NotContains(t, output, "Dapr and app processes stopped successfully")
		verifyCLIPIDNotExist(t, cliPID)
	})
}

func ensureAllAppsStartedWithRunTemplate(t *testing.T) {
	args := []string{
		"-f", "../testdata/run-template-files/dapr.yaml",
	}
	_, err := cmdRun("", args...)
	require.NoError(t, err, "run failed")
}

func tearDownTestSetup(t *testing.T) {
	// remove dapr installation after all tests in this function.
	must(t, cmdUninstall, "failed to uninstall Dapr")
	os.RemoveAll("../../apps/emit-metrics/.dapr/logs")
	os.RemoveAll("../../apps/processor/.dapr/logs")
}

func getCLIPID(t *testing.T) string {
	output, err := cmdList("json")
	require.NoError(t, err, "failed to list apps")
	result := []map[string]interface{}{}
	err = json.Unmarshal([]byte(output), &result)
	assert.Equal(t, 2, len(result))
	return fmt.Sprintf("%v", result[0]["cliPid"])
}

func verifyCLIPIDNotExist(t *testing.T, pid string) {
	time.Sleep(5 * time.Second)
	output, err := cmdList("")
	require.NoError(t, err, "failed to list apps")
	assert.NotContains(t, output, pid)
}
