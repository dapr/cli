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

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	startDaprRun(t, []int{3510}, func() { cmdStopWithAppID(appID) }, "-f", runFilePath)

	waitForAppHealthy(t, 60*time.Second, "test-workflow")

	// Purge any leftover workflow instances from previous test runs.
	purgeOut, purgeErr := cmdWorkflowPurge(appID, redisConnString, "--all")
	require.NoError(t, purgeErr, purgeOut)

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
}

func TestWorkflowRaiseEvent(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	startDaprRun(t, []int{3510}, func() { cmdStopWithAppID(appID) }, "-f", runFilePath)

	waitForAppHealthy(t, 60*time.Second, "test-workflow")
	purgeOut, purgeErr := cmdWorkflowPurge(appID, redisConnString, "--all")
	require.NoError(t, purgeErr, purgeOut)

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

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	startDaprRun(t, []int{3510}, func() { cmdStopWithAppID(appID) }, "-f", runFilePath)

	waitForAppHealthy(t, 60*time.Second, "test-workflow")

	purgeOut, purgeErr := cmdWorkflowPurge(appID, redisConnString, "--all")
	require.NoError(t, purgeErr, purgeOut)

	output, err := cmdWorkflowRun(appID, "SimpleWorkflow", "--instance-id=foo")
	require.NoError(t, err, output)

	// Wait for the workflow instance to reach a terminal state before
	// attempting rerun operations. Rerun requires the instance to be in a
	// terminal state (COMPLETED/FAILED/TERMINATED).
	require.Eventually(t, func() bool {
		out, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
		if err != nil {
			return false
		}
		var list []map[string]interface{}
		if err := json.Unmarshal([]byte(out), &list); err != nil {
			return false
		}
		for _, item := range list {
			if item["instanceID"] == "foo" {
				status, _ := item["runtimeStatus"].(string)
				return status == "COMPLETED" || status == "FAILED" || status == "TERMINATED"
			}
		}
		return false
	}, 60*time.Second, time.Second, "workflow instance 'foo' did not reach terminal state")

	t.Run("rerun from beginning", func(t *testing.T) {
		output, err := cmdWorkflowReRun(appID, "foo")
		require.NoError(t, err, output)
		assert.Contains(t, output, "Rerunning workflow instance")
	})

	t.Run("rerun with new instance ID", func(t *testing.T) {
		output, err := cmdWorkflowReRun(appID, "foo", "--new-instance-id", "bar")
		require.NoError(t, err, output)
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

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	startDaprRun(t, []int{3510}, func() { cmdStopWithAppID(appID) }, "-f", runFilePath)

	waitForAppHealthy(t, 60*time.Second, "test-workflow")
	purgeOut, purgeErr := cmdWorkflowPurge(appID, redisConnString, "--all")
	require.NoError(t, purgeErr, purgeOut)

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
		// Wait for workflows to reach terminal state after terminate.
		time.Sleep(2 * time.Second)

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

	t.Run("purge older than with filter status only purges matching status", func(t *testing.T) {
		// Create one workflow that will complete (SimpleWorkflow) and one that
		// will be terminated (LongWorkflow) so they have different statuses.
		output, err := cmdWorkflowRun(appID, "SimpleWorkflow",
			"--instance-id=filter-completed")
		require.NoError(t, err, output)

		output, err = cmdWorkflowRun(appID, "LongWorkflow",
			"--instance-id=filter-terminated")
		require.NoError(t, err, output)

		// Wait for SimpleWorkflow to complete.
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			out, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
			require.NoError(c, err)
			var list []map[string]interface{}
			require.NoError(c, json.Unmarshal([]byte(out), &list))
			for _, item := range list {
				if item["instanceID"] == "filter-completed" {
					assert.Equal(c, "COMPLETED", item["runtimeStatus"])
					return
				}
			}
			assert.Fail(c, "filter-completed not found")
		}, 30*time.Second, 500*time.Millisecond)

		// Terminate one so we have two different terminal statuses.
		_, err = cmdWorkflowTerminate(appID, "filter-terminated")
		require.NoError(t, err)

		// Wait for the terminate to take effect.
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			out, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
			require.NoError(c, err)
			var list []map[string]interface{}
			require.NoError(c, json.Unmarshal([]byte(out), &list))
			for _, item := range list {
				if item["instanceID"] == "filter-terminated" {
					assert.Equal(c, "TERMINATED", item["runtimeStatus"])
					return
				}
			}
			assert.Fail(c, "filter-terminated not found")
		}, 30*time.Second, 500*time.Millisecond)

		// Purge only COMPLETED instances older than 1s.
		output, err = cmdWorkflowPurge(appID, redisConnString,
			"--all-older-than", "1s", "--all-filter-status", "COMPLETED")
		require.NoError(t, err, output)
		assert.Contains(t, output, `Purged workflow instance "filter-completed"`)
		assert.NotContains(t, output, "filter-terminated")

		// Verify filter-terminated still exists.
		output, err = cmdWorkflowList(appID, "-o", "json", redisConnString)
		require.NoError(t, err, output)
		assert.NotContains(t, output, "filter-completed")
		assert.Contains(t, output, "filter-terminated")

		// Clean up the remaining instance.
		t.Cleanup(func() {
			_, err := cmdWorkflowPurge(appID, redisConnString, "filter-terminated")
			assert.NoError(t, err)
		})
	})

	t.Run("purge older than with filter status TERMINATED", func(t *testing.T) {
		output, err := cmdWorkflowRun(appID, "SimpleWorkflow",
			"--instance-id=fs-completed")
		require.NoError(t, err, output)

		output, err = cmdWorkflowRun(appID, "LongWorkflow",
			"--instance-id=fs-terminated")
		require.NoError(t, err, output)

		// Wait for SimpleWorkflow to complete.
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			out, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
			require.NoError(c, err)
			var list []map[string]interface{}
			require.NoError(c, json.Unmarshal([]byte(out), &list))
			for _, item := range list {
				if item["instanceID"] == "fs-completed" {
					assert.Equal(c, "COMPLETED", item["runtimeStatus"])
					return
				}
			}
			assert.Fail(c, "fs-completed not found")
		}, 30*time.Second, 500*time.Millisecond)

		_, err = cmdWorkflowTerminate(appID, "fs-terminated")
		require.NoError(t, err)

		// Wait for the terminate to take effect.
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			out, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
			require.NoError(c, err)
			var list []map[string]interface{}
			require.NoError(c, json.Unmarshal([]byte(out), &list))
			for _, item := range list {
				if item["instanceID"] == "fs-terminated" {
					assert.Equal(c, "TERMINATED", item["runtimeStatus"])
					return
				}
			}
			assert.Fail(c, "fs-terminated not found")
		}, 30*time.Second, 500*time.Millisecond)

		// Purge only TERMINATED instances older than 1s.
		output, err = cmdWorkflowPurge(appID, redisConnString,
			"--all-older-than", "1s", "--all-filter-status", "TERMINATED")
		require.NoError(t, err, output)
		assert.Contains(t, output, `Purged workflow instance "fs-terminated"`)
		assert.NotContains(t, output, "fs-completed")

		// Verify fs-completed still exists.
		output, err = cmdWorkflowList(appID, "-o", "json", redisConnString)
		require.NoError(t, err, output)
		assert.Contains(t, output, "fs-completed")
		assert.NotContains(t, output, "fs-terminated")

		// Clean up.
		t.Cleanup(func() {
			_, err := cmdWorkflowPurge(appID, redisConnString, "fs-completed")
			assert.NoError(t, err)
		})
	})

	t.Run("all-filter-status without all-older-than errors", func(t *testing.T) {
		_, err := cmdWorkflowPurge(appID, redisConnString,
			"--all-filter-status", "COMPLETED")
		require.Error(t, err)
	})

	t.Run("all-filter-status with invalid value errors", func(t *testing.T) {
		_, err := cmdWorkflowPurge(appID, redisConnString,
			"--all-older-than", "1s", "--all-filter-status", "INVALID")
		require.Error(t, err)
	})

	t.Run("all-filter-status with all flag errors", func(t *testing.T) {
		_, err := cmdWorkflowPurge(appID, redisConnString,
			"--all", "--all-filter-status", "COMPLETED")
		require.Error(t, err)
	})

	t.Run("also purge scheduler", func(t *testing.T) {
		output, err := cmdWorkflowRun(appID, "EventWorkflow",
			"--instance-id=also-sched")
		require.NoError(t, err)

		// Wait for scheduler entries to appear while workflow is still running.
		require.Eventually(t, func() bool {
			output, err := cmdSchedulerList()
			if err != nil {
				return false
			}
			return len(strings.Split(output, "\n")) > 2
		}, 30*time.Second, time.Second, "expected scheduler entries to appear")

		output, err = cmdWorkflowTerminate(appID, "also-sched")
		require.NoError(t, err, output)

		output, err = cmdWorkflowPurge(appID, "also-sched")
		require.NoError(t, err, output)

		require.Eventually(t, func() bool {
			output, err := cmdSchedulerList()
			if err != nil {
				return false
			}
			return len(strings.Split(output, "\n")) == 2
		}, 30*time.Second, time.Second, "expected scheduler entries to be purged")
	})
}

func TestWorkflowFilters(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	startDaprRun(t, []int{3510}, func() { cmdStopWithAppID(appID) }, "-f", runFilePath)

	waitForAppHealthy(t, 60*time.Second, "test-workflow")
	purgeOut, purgeErr := cmdWorkflowPurge(appID, redisConnString, "--all")
	require.NoError(t, purgeErr, purgeOut)

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
		assert.NotContains(t, output, "simple-1")
		assert.NotContains(t, output, "long-1")
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

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	startDaprRun(t, []int{3510}, func() { cmdStopWithAppID(appID) }, "-f", runFilePath)

	waitForAppHealthy(t, 60*time.Second, "test-workflow")
	purgeOut, purgeErr := cmdWorkflowPurge(appID, redisConnString, "--all")
	require.NoError(t, purgeErr, purgeOut)

	t.Run("parent child workflow", func(t *testing.T) {
		input := `{"test": "parent-child", "value": 42}`
		output, err := cmdWorkflowRun(appID, "ParentWorkflow", "--input", input, "--instance-id=parent-1")
		require.NoError(t, err, output)

		// Poll until the parent workflow and child workflows appear.
		require.Eventually(t, func() bool {
			out, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
			if err != nil {
				return false
			}
			var list []map[string]interface{}
			if err := json.Unmarshal([]byte(out), &list); err != nil {
				return false
			}
			var parentFound bool
			var childCount int
			for _, item := range list {
				if item["instanceID"] == "parent-1" {
					parentFound = true
				}
				if name, ok := item["name"].(string); ok && name == "ChildWorkflow" {
					childCount++
				}
			}
			return parentFound && childCount >= 2
		}, 30*time.Second, time.Second, "parent workflow and children did not appear")

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

		// Poll until recursive child workflows appear.
		require.Eventually(t, func() bool {
			out, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
			if err != nil {
				return false
			}
			var list []map[string]interface{}
			if err := json.Unmarshal([]byte(out), &list); err != nil {
				return false
			}
			count := 0
			for _, item := range list {
				if name, ok := item["name"].(string); ok && name == "RecursiveChildWorkflow" {
					count++
				}
			}
			return count >= 2
		}, 30*time.Second, time.Second, "recursive child workflows did not appear")

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

		// Poll until fan-out child workflows appear.
		require.Eventually(t, func() bool {
			out, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
			if err != nil {
				return false
			}
			var list []map[string]interface{}
			if err := json.Unmarshal([]byte(out), &list); err != nil {
				return false
			}
			count := 0
			for _, item := range list {
				if name, ok := item["name"].(string); ok && name == "ChildWorkflow" {
					count++
				}
			}
			return count >= parallelCount
		}, 30*time.Second, time.Second, "fan-out child workflows did not appear")

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
		output, err := cmdWorkflowRun(appID, "ParentWorkflow", "--input", `{"fail": true}`, "--instance-id=parent-fail-1")
		require.NoError(t, err, output)

		// Poll until the parent workflow reaches a terminal state.
		// On slow CI runners the workflow may still be RUNNING after 5s.
		require.Eventually(t, func() bool {
			out, err := cmdWorkflowList(appID, redisConnString, "-o", "json")
			if err != nil {
				return false
			}
			var list []map[string]interface{}
			if err := json.Unmarshal([]byte(out), &list); err != nil {
				return false
			}
			for _, item := range list {
				if item["instanceID"] == "parent-fail-1" {
					status, _ := item["runtimeStatus"].(string)
					return status == "COMPLETED" || status == "FAILED"
				}
			}
			return false
		}, 30*time.Second, time.Second, "parent-fail-1 workflow did not reach terminal state")
	})
}

func TestWorkflowHistory(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	startDaprRun(t, []int{3510}, func() { cmdStopWithAppID(appID) }, "-f", runFilePath)

	waitForAppHealthy(t, 60*time.Second, "test-workflow")
	purgeOut, purgeErr := cmdWorkflowPurge(appID, redisConnString, "--all")
	require.NoError(t, purgeErr, purgeOut)

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

	t.Run("timer origin createTimer", func(t *testing.T) {
		// WTimer calls ctx.CreateTimer which produces origin=createTimer.
		_, err := cmdWorkflowRun(appID, "WTimer", "--instance-id=timer-origin-test")
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			out, err := cmdWorkflowHistory(appID, "timer-origin-test")
			return err == nil && strings.Contains(out, "origin=createTimer")
		}, 10*time.Second, 200*time.Millisecond)

		output, err := cmdWorkflowHistory(appID, "timer-origin-test", "-o", "json")
		require.NoError(t, err)

		var history []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &history))

		found := false
		for _, ev := range history {
			if ev["type"] == "TimerCreated" {
				if attrs, ok := ev["attrs"].(string); ok {
					assert.Contains(t, attrs, "origin=createTimer")
					found = true
				}
			}
		}
		assert.True(t, found, "expected TimerCreated event with origin=createTimer in attrs")
	})

	t.Run("timer origin externalEvent", func(t *testing.T) {
		// EventWorkflow calls ctx.WaitForExternalEvent("test-event", time.Hour)
		// which produces origin=externalEvent(test-event).
		_, err := cmdWorkflowRun(appID, "EventWorkflow", "--instance-id=event-origin-test")
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			out, err := cmdWorkflowHistory(appID, "event-origin-test")
			return err == nil && strings.Contains(out, "origin=externalEvent(test-event)")
		}, 10*time.Second, 200*time.Millisecond)

		output, err := cmdWorkflowHistory(appID, "event-origin-test", "-o", "json")
		require.NoError(t, err)

		var history []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &history))

		found := false
		for _, ev := range history {
			if ev["type"] == "TimerCreated" {
				if attrs, ok := ev["attrs"].(string); ok {
					assert.Contains(t, attrs, "origin=externalEvent")
					assert.Contains(t, attrs, "eventName=test-event")
					found = true
				}
			}
		}
		assert.True(t, found, "expected TimerCreated event with origin=externalEvent in attrs")
	})

	t.Run("timer origin activityRetry", func(t *testing.T) {
		// ActivityRetryWorkflow calls a failing activity with retry policy,
		// producing TimerCreated events with origin=activityRetry.
		_, err := cmdWorkflowRun(appID, "ActivityRetryWorkflow", "--instance-id=activity-retry-origin-test")
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			out, err := cmdWorkflowHistory(appID, "activity-retry-origin-test")
			return err == nil && strings.Contains(out, "origin=activityRetry(")
		}, 15*time.Second, 250*time.Millisecond)

		output, err := cmdWorkflowHistory(appID, "activity-retry-origin-test", "-o", "json")
		require.NoError(t, err)

		var history []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &history))

		found := false
		for _, ev := range history {
			if ev["type"] == "TimerCreated" {
				if attrs, ok := ev["attrs"].(string); ok && strings.Contains(attrs, "origin=activityRetry") {
					assert.Contains(t, attrs, "taskExecId=")
					found = true
				}
			}
		}
		assert.True(t, found, "expected TimerCreated event with origin=activityRetry in attrs")
	})

	t.Run("timer origin childWorkflowRetry", func(t *testing.T) {
		// ChildWorkflowRetryWorkflow calls a failing child workflow with retry
		// policy, producing TimerCreated events with origin=childWorkflowRetry.
		_, err := cmdWorkflowRun(appID, "ChildWorkflowRetryWorkflow", "--instance-id=child-retry-origin-test")
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			out, err := cmdWorkflowHistory(appID, "child-retry-origin-test")
			return err == nil && strings.Contains(out, "origin=childWorkflowRetry(")
		}, 30*time.Second, 500*time.Millisecond)

		output, err := cmdWorkflowHistory(appID, "child-retry-origin-test", "-o", "json")
		require.NoError(t, err)

		var history []map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(output), &history))

		found := false
		for _, ev := range history {
			if ev["type"] == "TimerCreated" {
				if attrs, ok := ev["attrs"].(string); ok && strings.Contains(attrs, "origin=childWorkflowRetry") {
					assert.Contains(t, attrs, "instanceId=")
					found = true
				}
			}
		}
		assert.True(t, found, "expected TimerCreated event with origin=childWorkflowRetry in attrs")
	})
}

func TestWorkflowSuspendResume(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	startDaprRun(t, []int{3510}, func() { cmdStopWithAppID(appID) }, "-f", runFilePath)

	waitForAppHealthy(t, 60*time.Second, "test-workflow")
	purgeOut, purgeErr := cmdWorkflowPurge(appID, redisConnString, "--all")
	require.NoError(t, purgeErr, purgeOut)

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

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	startDaprRun(t, []int{3510}, func() { cmdStopWithAppID(appID) }, "-f", runFilePath)

	waitForAppHealthy(t, 60*time.Second, "test-workflow")
	purgeOut, purgeErr := cmdWorkflowPurge(appID, redisConnString, "--all")
	require.NoError(t, purgeErr, purgeOut)

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
