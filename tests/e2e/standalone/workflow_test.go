//go:build !windows && (e2e || template)

/*
Copyright 2025 The Dapr Authors
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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	redisConnString = "--connection-string=redis://127.0.0.1:6379"
)

func TestWorkflowList(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	t.Cleanup(func() {
		cmdStopWithAppID(appID)
		waitAppsToBeStopped()
	})
	args := []string{"-f", runFilePath}

	go func() {
		o, _ := cmdRun("", args...)
		t.Log(o)
	}()

	time.Sleep(time.Second * 5)
	output, err := cmdWorkflowList(appID, redisConnString)
	require.NoError(t, err)
	assert.Equal(t, `❌  No workflow found in namespace "default" for app ID "test-workflow"
`, output)

	_, err = cmdWorkflowRun(appID, "LongWorkflow", "--instance-id=foo")
	require.NoError(t, err, output)

	t.Run("terminate workflow", func(t *testing.T) {
		output, err := cmdWorkflowTerminate(appID, "foo")
		require.NoError(t, err)
		assert.Contains(t, output, "terminated successfully")
	})

	t.Run("verify terminated state", func(t *testing.T) {
		output, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
		require.NoError(t, err, output)

		var list []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &list))

		found := false
		for _, item := range list {
			if item["instanceID"] == "foo" {
				assert.Equal(t, "TERMINATED", item["runtimeStatus"])
				found = true
				break
			}
		}
		assert.True(t, found, "Workflow instance not found")
	})

	t.Run("terminate with output", func(t *testing.T) {
		output, err := cmdWorkflowRun(appID, "LongWorkflow", "--instance-id=bar")
		require.NoError(t, err)

		outputData := `{"reason": "test termination", "code": 123}`
		output, err = cmdWorkflowTerminate(appID, "bar", "-o", outputData)
		require.NoError(t, err)
		assert.Contains(t, output, "terminated successfully")
	})
}

func TestWorkflowRaiseEvent(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	t.Cleanup(func() {
		cmdStopWithAppID(appID)
		waitAppsToBeStopped()
	})
	args := []string{"-f", runFilePath}

	go func() {
		o, _ := cmdRun("", args...)
		t.Log(o)
	}()

	time.Sleep(time.Second * 5)
	output, err := cmdWorkflowRun(appID, "EventWorkflow", "--instance-id=foo")
	require.NoError(t, err, output)

	t.Run("raise event", func(t *testing.T) {
		output, err := cmdWorkflowRaiseEvent(appID, "foo/test-event")
		require.NoError(t, err)
		assert.Contains(t, output, "raised event")
		assert.Contains(t, output, "successfully")

		time.Sleep(time.Second)

		output, err = cmdWorkflowList(appID, redisConnString, "-o", "json")
		require.NoError(t, err, output)

		var list []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &list))

		found := false
		for _, item := range list {
			if item["instanceID"] == "foo" {
				assert.Equal(t, "COMPLETED", item["runtimeStatus"])
				found = true
				break
			}
		}
		assert.True(t, found, "Workflow instance not found")
	})

	t.Run("raise event with input", func(t *testing.T) {
		output, err := cmdWorkflowRun(appID, "EventWorkflow", "--instance-id=bar")
		require.NoError(t, err)

		input := `{"eventData": "test data", "value": 456}`
		output, err = cmdWorkflowRaiseEvent(appID, "bar/test-event", "--input", input)
		require.NoError(t, err)
		assert.Contains(t, output, "raised event")

		time.Sleep(time.Second)

		output, err = cmdWorkflowList(appID, redisConnString, "-o", "json")
		require.NoError(t, err, output)

		var list []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &list))

		found := false
		for _, item := range list {
			if item["instanceID"] == "foo" {
				assert.Equal(t, "COMPLETED", item["runtimeStatus"])
				found = true
				break
			}
		}
		assert.True(t, found, "Workflow instance not found")
	})
}

func TestWorkflowReRun(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	t.Cleanup(func() {
		cmdStopWithAppID(appID)
		waitAppsToBeStopped()
	})
	args := []string{"-f", runFilePath}

	go func() {
		o, _ := cmdRun("", args...)
		t.Log(o)
	}()

	time.Sleep(time.Second * 5)

	output, err := cmdWorkflowRun(appID, "SimpleWorkflow", "--instance-id=foo")
	require.NoError(t, err, output)

	time.Sleep(3 * time.Second)

	t.Run("rerun from beginning", func(t *testing.T) {
		output, err := cmdWorkflowReRun(appID, "foo")
		require.NoError(t, err)
		assert.Contains(t, output, "Rerunning workflow instance")
	})

	t.Run("rerun with new instance ID", func(t *testing.T) {
		output, err := cmdWorkflowReRun(appID, "foo", "--new-instance-id", "bar")
		require.NoError(t, err)
		assert.Contains(t, output, "bar")
	})

	t.Run("rerun from specific event", func(t *testing.T) {
		output, err := cmdWorkflowReRun(appID, "foo", "-e", "1")
		require.NoError(t, err)
		assert.Contains(t, output, "Rerunning workflow instance")
	})

	t.Run("rerun with new input", func(t *testing.T) {
		input := `{"rerun": true, "data": "new input"}`
		output, err := cmdWorkflowReRun(appID, "foo", "--input", input)
		require.NoError(t, err)
		assert.Contains(t, output, "Rerunning workflow instance")
	})
}

func TestWorkflowPurge(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	t.Cleanup(func() {
		cmdStopWithAppID(appID)
		waitAppsToBeStopped()
	})
	args := []string{"-f", runFilePath}

	go func() {
		o, _ := cmdRun("", args...)
		t.Log(o)
	}()

	time.Sleep(5 * time.Second)

	for i := 0; i < 3; i++ {
		output, err := cmdWorkflowRun(appID, "SimpleWorkflow",
			"--instance-id=purge-test-"+strconv.Itoa(i))
		require.NoError(t, err, output)
	}

	time.Sleep(5 * time.Second)

	_, err := cmdWorkflowTerminate(appID, "purge-test-0")
	require.NoError(t, err)

	t.Run("purge single instance", func(t *testing.T) {
		output, err := cmdWorkflowPurge(appID, "purge-test-0")
		require.NoError(t, err)
		assert.Contains(t, output, "Purged")

		output, err = cmdWorkflowList(appID, "-o", "json", redisConnString)
		require.NoError(t, err)
		assert.NotContains(t, output, "purge-test-0")
	})

	t.Run("purge multiple instances", func(t *testing.T) {
		_, _ = cmdWorkflowTerminate(appID, "purge-test-1")
		_, _ = cmdWorkflowTerminate(appID, "purge-test-2")
		time.Sleep(1 * time.Second)

		output, err := cmdWorkflowPurge(appID, "purge-test-1", "purge-test-2")
		require.NoError(t, err)
		assert.Contains(t, output, "Purged")
	})

	t.Run("purge all terminal", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			output, err := cmdWorkflowRun(appID, "SimpleWorkflow",
				"--instance-id=purge-all-"+strconv.Itoa(i))
			require.NoError(t, err, output)
			_, _ = cmdWorkflowTerminate(appID, "purge-all-"+strconv.Itoa(i))
		}

		output, err := cmdWorkflowPurge(appID, redisConnString, "--all")
		require.NoError(t, err, output)
		assert.Contains(t, output, `Purged workflow instance "purge-all-1"`)
		assert.Contains(t, output, `Purged workflow instance "purge-all-0"`)

		output, err = cmdWorkflowList(appID, "-o", "json", redisConnString)
		require.NoError(t, err)
		assert.NotContains(t, output, "purge-all-0")
		assert.NotContains(t, output, "purge-all-1")
	})

	t.Run("purge older than duration", func(t *testing.T) {
		output, err := cmdWorkflowRun(appID, "SimpleWorkflow",
			"--instance-id=purge-older")
		require.NoError(t, err)

		time.Sleep(5 * time.Second)

		output, err = cmdWorkflowPurge(appID, redisConnString, "--all-older-than", "1s")
		require.NoError(t, err, output)
		assert.Contains(t, output, "Purging 1 workflow instance(s)")
		assert.Contains(t, output, `Purged workflow instance "purge-older"`)

		output, err = cmdWorkflowList(appID, "-o", "json", redisConnString)
		require.NoError(t, err, output)
		assert.NotContains(t, output, "purge-older")
	})
}

func TestWorkflowFilters(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	t.Cleanup(func() {
		cmdStopWithAppID(appID)
		waitAppsToBeStopped()
	})
	args := []string{"-f", runFilePath}

	go func() {
		o, _ := cmdRun("", args...)
		t.Log(o)
	}()

	time.Sleep(5 * time.Second)

	_, _ = cmdWorkflowRun(appID, "SimpleWorkflow", "--instance-id=simple-1")
	_, _ = cmdWorkflowRun(appID, "LongWorkflow", "--instance-id=long-1")
	output, err := cmdWorkflowRun(appID, "EventWorkflow", "--instance-id=suspend-test")
	require.NoError(t, err, output)

	time.Sleep(2 * time.Second)
	_, _ = cmdWorkflowSuspend(appID, "suspend-test")

	t.Run("filter by status", func(t *testing.T) {
		output, err := cmdWorkflowList(appID, redisConnString, "--filter-status", "SUSPENDED")
		require.NoError(t, err)
		assert.Contains(t, output, "suspend-test")
	})

	t.Run("filter by name", func(t *testing.T) {
		output, err := cmdWorkflowList(appID, redisConnString, "--filter-name", "SimpleWorkflow")
		require.NoError(t, err)
		lines := strings.Split(output, "\n")

		for i, line := range lines {
			if i == 0 || strings.TrimSpace(line) == "" {
				continue
			}
			assert.Contains(t, line, "SimpleWorkflow")
		}
	})

	t.Run("filter by max age", func(t *testing.T) {
		output, err := cmdWorkflowList(appID, redisConnString, "--filter-max-age", "10s")
		require.NoError(t, err)
		assert.NotEmpty(t, output)

		output, err = cmdWorkflowList(appID, redisConnString, "--filter-max-age", "0s")
		require.NoError(t, err)
		lines := strings.Split(output, "\n")
		assert.LessOrEqual(t, len(lines), 2)
	})
}

func TestWorkflowChildCalls(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	t.Cleanup(func() {
		cmdStopWithAppID(appID)
		waitAppsToBeStopped()
	})
	args := []string{"-f", runFilePath}

	go func() {
		o, _ := cmdRun("", args...)
		t.Log(o)
	}()

	time.Sleep(5 * time.Second)

	t.Run("parent child workflow", func(t *testing.T) {
		input := `{"test": "parent-child", "value": 42}`
		output, err := cmdWorkflowRun(appID, "ParentWorkflow", "--input", input, "--instance-id=parent-1")
		require.NoError(t, err, output)

		time.Sleep(5 * time.Second)

		output, err = cmdWorkflowList(appID, redisConnString, "-o", "json")
		require.NoError(t, err)

		var list []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &list))

		var parentFound bool
		var childCount int
		for _, item := range list {
			if item["instanceID"] == "parent-1" {
				parentFound = true
				assert.Equal(t, "ParentWorkflow", item["name"])
			}
			if name, ok := item["name"].(string); ok && name == "ChildWorkflow" {
				childCount++
			}
		}
		assert.True(t, parentFound, "Parent workflow not found")
		assert.GreaterOrEqual(t, childCount, 2, "Expected at least 2 child workflows")
	})

	t.Run("nested child workflows", func(t *testing.T) {
		output, err := cmdWorkflowRun(appID, "NestedParentWorkflow", "--instance-id=nested-parent")
		require.NoError(t, err)

		time.Sleep(6 * time.Second)

		output, err = cmdWorkflowList(appID, redisConnString, "-o", "json")
		require.NoError(t, err)

		var list []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &list))

		var recursiveCount int
		for _, item := range list {
			if name, ok := item["name"].(string); ok && name == "RecursiveChildWorkflow" {
				recursiveCount++
			}
		}
		assert.GreaterOrEqual(t, recursiveCount, 2, "Expected multiple recursive child workflows")
	})

	t.Run("fan out workflow", func(t *testing.T) {
		parallelCount := 5
		input := fmt.Sprintf(`{"parallelCount": %d, "data": {"test": "fanout"}}`, parallelCount)
		output, err := cmdWorkflowRun(appID, "FanOutWorkflow", "--input", input, "--instance-id=fanout-1")
		require.NoError(t, err)

		time.Sleep(5 * time.Second)

		output, err = cmdWorkflowList(appID, redisConnString, "-o", "json")
		require.NoError(t, err)

		var list []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &list))

		var fanOutChildren int
		for _, item := range list {
			if name, ok := item["name"].(string); ok && name == "ChildWorkflow" {
				fanOutChildren++
			}
		}
		assert.GreaterOrEqual(t, fanOutChildren, parallelCount, "Expected at least %d child workflows from fan-out", parallelCount)
	})

	t.Run("child workflow failure handling", func(t *testing.T) {
		output, err := cmdWorkflowRun(appID, "ParentWorkflow", "--input", `{"fail": true}`, "--instance-id=parent-1")
		require.NoError(t, err, output)

		time.Sleep(5 * time.Second)

		output, err = cmdWorkflowList(appID, redisConnString, "-o", "json")
		require.NoError(t, err)

		var list []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &list))

		for _, item := range list {
			if item["instanceID"] == "parent-1" {
				status := item["runtimeStatus"].(string)
				assert.Contains(t, []string{"COMPLETED", "FAILED"}, status)
				break
			}
		}
	})
}

func TestWorkflowHistory(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	t.Cleanup(func() {
		cmdStopWithAppID(appID)
		waitAppsToBeStopped()
	})
	args := []string{"-f", runFilePath}

	go func() {
		o, _ := cmdRun("", args...)
		t.Log(o)
	}()

	// Wait and create a workflow
	time.Sleep(5 * time.Second)
	output, err := cmdWorkflowRun(appID, "SimpleWorkflow", "--instance-id=history-test")
	require.NoError(t, err, output)

	// Wait for workflow to have some history
	time.Sleep(2 * time.Second)

	t.Run("get history", func(t *testing.T) {
		output, err := cmdWorkflowHistory(appID, "history-test")
		require.NoError(t, err)
		lines := strings.Split(output, "\n")

		// Should have headers and at least one history entry
		assert.GreaterOrEqual(t, len(lines), 2)

		headers := strings.Fields(lines[0])
		assert.Contains(t, headers, "TYPE")
		assert.Contains(t, headers, "ELAPSED")
	})

	t.Run("get history json", func(t *testing.T) {
		output, err := cmdWorkflowHistory(appID, "history-test", "-o", "json")
		require.NoError(t, err)

		var history []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &history))
		assert.GreaterOrEqual(t, len(history), 1)
	})
}

func TestWorkflowSuspendResume(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	t.Cleanup(func() {
		cmdStopWithAppID(appID)
		waitAppsToBeStopped()
	})
	args := []string{"-f", runFilePath}

	go func() {
		o, _ := cmdRun("", args...)
		t.Log(o)
	}()

	// Wait and create a long-running workflow
	time.Sleep(5 * time.Second)
	output, err := cmdWorkflowRun(appID, "LongWorkflow", "--instance-id=suspend-resume-test")
	require.NoError(t, err, output)

	t.Run("suspend workflow", func(t *testing.T) {
		output, err := cmdWorkflowSuspend(appID, "suspend-resume-test")
		require.NoError(t, err, output)
		assert.Contains(t, output, "Workflow 'suspend-resume-test' suspended successfully")
	})

	t.Run("verify suspended state", func(t *testing.T) {
		output, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
		require.NoError(t, err, output)

		var list []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &list))

		found := false
		for _, item := range list {
			if item["instanceID"] == "suspend-resume-test" {
				assert.Equal(t, "SUSPENDED", item["runtimeStatus"])
				found = true
				break
			}
		}
		assert.True(t, found, "Workflow instance not found")
	})

	t.Run("resume workflow", func(t *testing.T) {
		output, err := cmdWorkflowResume(appID, "suspend-resume-test")
		require.NoError(t, err)
		assert.Contains(t, output, "Workflow 'suspend-resume-test' resumed successfully")
	})

	t.Run("verify resumed state", func(t *testing.T) {
		time.Sleep(1 * time.Second)
		output, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
		require.NoError(t, err)

		var list []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &list))

		found := false
		for _, item := range list {
			if item["instanceID"] == "suspend-resume-test" {
				assert.NotEqual(t, "SUSPENDED", item["runtimeStatus"])
				found = true
				break
			}
		}
		assert.True(t, found, "Workflow instance not found")
	})
}

func TestWorkflowTerminate(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	t.Cleanup(func() {
		cmdStopWithAppID(appID)
		waitAppsToBeStopped()
	})
	args := []string{"-f", runFilePath}

	go func() {
		o, _ := cmdRun("", args...)
		t.Log(o)
	}()

	// Wait and create a workflow for testing
	time.Sleep(5 * time.Second)
	output, err := cmdWorkflowRun(appID, "LongWorkflow", "--instance-id=terminate-test")
	require.NoError(t, err, output)

	t.Run("terminate workflow", func(t *testing.T) {
		output, err := cmdWorkflowTerminate(appID, "terminate-test")
		require.NoError(t, err)
		assert.Contains(t, output, "terminated successfully")
	})

	t.Run("verify terminated state", func(t *testing.T) {
		output, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
		require.NoError(t, err)

		var list []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &list))

		found := false
		for _, item := range list {
			if item["instanceID"] == "terminate-test" {
				assert.Equal(t, "TERMINATED", item["runtimeStatus"])
				found = true
				break
			}
		}
		assert.True(t, found, "Workflow instance not found")
	})

	t.Run("terminate with output", func(t *testing.T) {
		// Create another workflow
		output, err := cmdWorkflowRun(appID, "LongWorkflow", "--instance-id=terminate-output-test")
		require.NoError(t, err, output)

		outputData := `{"reason": "test termination", "code": 123}`
		output, err = cmdWorkflowTerminate(appID, "terminate-output-test", "-o", outputData)
		require.NoError(t, err)
		assert.Contains(t, output, "terminated successfully")
	})
}
