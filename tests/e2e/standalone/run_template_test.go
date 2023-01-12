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
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type AppTestOutput struct {
	appID           string
	appLogContents  []string
	daprdLogContent []string
	baseLogDirPath  string
}

func TestRunWithTemplateFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})
	t.Run(fmt.Sprintf("check run with -f file"), func(t *testing.T) {
		// This test is dependent on run template file in ../testdata/multipleapps/dapr.yaml
		args := []string{
			"-f", "../testdata/dapr.yaml",
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		output, err := cmdRunWithContext(ctx, "", args...)
		t.Logf(output)
		require.NoError(t, err, "run failed")
		// Deterministic output for template file, so we can assert line by line
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 4, "expected at least 4 lines in output of starting two apps")
		assert.Contains(t, lines[0], "Started Dapr with app id \"processor\". HTTP Port: 3510.")
		assert.Contains(t, lines[1], "Writing log files to directory")
		assert.Contains(t, lines[1], "tests/apps/processor/.dapr/logs")
		assert.Contains(t, lines[2], "Started Dapr with app id \"emit-metrics\". HTTP Port: 3511.")
		assert.Contains(t, lines[3], "Writing log files to directory")
		assert.Contains(t, lines[3], "tests/apps/emit-metrics/.dapr/logs")
		assert.Contains(t, output, "Received signal to stop Dapr and app processes. Shutting down Dapr and app processes.")
		appTestOutput := AppTestOutput{
			appID:          "processor",
			baseLogDirPath: "../../apps/processor/.dapr/logs",
			appLogContents: []string{
				"Received metrics:  {3}",
			},
			daprdLogContent: []string{
				"http server is running on port 3510",
				"You're up and running! Dapr logs will appear here.",
			},
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
		appTestOutput = AppTestOutput{
			appID:          "emit-metrics",
			baseLogDirPath: "../../apps/emit-metrics/.dapr/logs",
			appLogContents: []string{
				"DAPR_HTTP_PORT set to  3511",
				"Metrics with ID 3 sent",
			},
			daprdLogContent: []string{
				"termination signal received: shutting down",
				"Exited Dapr successfully",
				"Exited App successfully",
			},
		}
		assertLogOutputForRunTemplateExec(t, appTestOutput)
	})
}

func assertLogOutputForRunTemplateExec(t *testing.T, appTestOutput AppTestOutput) {
	daprdLogPath := filepath.Join(appTestOutput.baseLogDirPath, "daprd.log")
	appLogPath := filepath.Join(appTestOutput.baseLogDirPath, "app.log")
	assert.FileExists(t, daprdLogPath, "daprd log must exist")
	assert.FileExists(t, appLogPath, "app log must exist")
	fileContents, err := ioutil.ReadFile(daprdLogPath)
	assert.NoError(t, err, "failed to read daprd log")
	contentString := string(fileContents)
	for _, line := range appTestOutput.daprdLogContent {
		assert.Contains(t, contentString, line, "expected logline to be present")
	}
	fileContents, err = ioutil.ReadFile(appLogPath)
	assert.NoError(t, err, "failed to read app log")
	contentString = string(fileContents)
	for _, line := range appTestOutput.appLogContents {
		assert.Contains(t, contentString, line, "expected logline to be present")
	}
}
