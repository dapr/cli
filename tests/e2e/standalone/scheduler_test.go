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
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dapr/cli/pkg/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// countSchedulerEntries parses the tabular output from `dapr scheduler list`
// and returns the number of data rows (skipping the header and empty lines).
// This avoids hard-coding total line counts that break when the output format
// changes (e.g. extra trailing newlines or header adjustments).
func countSchedulerEntries(output string) int {
	count := 0
	for i, line := range strings.Split(output, "\n") {
		if i == 0 { // skip header
			continue
		}
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func TestSchedulerList(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping scheduler tests in slim mode")
	}

	// Reinstall Dapr to get a fresh scheduler container. Without this,
	// stale workflow registrations from previous tests cause
	// wf.StartWorker to hang when reconnecting with the same types/IDs.
	cmdUninstall()
	ensureDaprInstallation(t)

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	startDaprRunRetry(t, []int{3510}, func() { cmdStopWithRunTemplate(runFilePath) }, "-f", runFilePath)

	// On slow CI runners, the first dapr run attempt may fail to register
	// workflows (only jobs + reminders appear). startDaprRunRetry retries
	// in the background, but the retry can take 30-40s. Use 120s to
	// accommodate the retry delay.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.GreaterOrEqual(c, countSchedulerEntries(output), 8)
	}, 240*time.Second, time.Second)

	t.Run("short", func(t *testing.T) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		lines := strings.Split(output, "\n")
		require.Equal(t, 8, countSchedulerEntries(output))

		require.Equal(t, []string{
			"NAME",
			"BEGIN",
			"COUNT",
			"LAST",
			"TRIGGER",
		}, strings.Fields(lines[0]))

		// use map for order-independent checking
		schedulerCounts := make(map[string]int)
		for _, line := range lines[1:] {
			if strings.TrimSpace(line) == "" {
				continue
			}
			fields := strings.Fields(line)
			// Need at least 3 fields to access NAME (fields[0]) and COUNT (fields[2])
			// Header is: NAME, BEGIN, COUNT, LAST, TRIGGER
			if len(fields) < 3 {
				continue
			}
			name := fields[0]
			count, err := strconv.Atoi(fields[2])
			if err != nil {
				t.Logf("skipping line with invalid count: %s (error: %v)", line, err)
				continue
			}
			schedulerCounts[name] = count
		}

		// Check actor reminders (count should be 1)
		expActorNames := []string{
			"actor/myactortype/actorid1/test1",
			"actor/myactortype/actorid2/test2",
		}
		for _, name := range expActorNames {
			count, exists := schedulerCounts[name]
			require.True(t, exists, "expected actor reminder %s not found", name)
			assert.Equal(t, 1, count, "actor reminder %s should have count 1", name)
		}

		// Check app jobs (count should be 1)
		expAppNames := []string{
			"app/test-scheduler/test1",
			"app/test-scheduler/test2",
		}
		for _, name := range expAppNames {
			count, exists := schedulerCounts[name]
			require.True(t, exists, "expected app job %s not found", name)
			assert.Equal(t, 1, count, "app job %s should have count 1", name)
		}

		// Check activity items (count should be 0)
		expActivityNames := []string{
			"activity/test-scheduler/xyz1::0::1",
			"activity/test-scheduler/xyz2::0::1",
		}
		for _, name := range expActivityNames {
			count, exists := schedulerCounts[name]
			require.True(t, exists, "expected activity %s not found", name)
			assert.Equal(t, 0, count, "activity %s should have count 0", name)
		}

		expWorkflowPrefixes := []string{
			"workflow/test-scheduler/abc1",
			"workflow/test-scheduler/abc2",
		}
		foundWorkflows := 0
		for name := range schedulerCounts {
			for _, prefix := range expWorkflowPrefixes {
				if strings.HasPrefix(name, prefix) {
					foundWorkflows++
					break
				}
			}
		}
		assert.Equal(t, len(expWorkflowPrefixes), foundWorkflows, "expected %d workflow items", len(expWorkflowPrefixes))
	})

	t.Run("wide", func(t *testing.T) {
		output, err := cmdSchedulerList("-o", "wide")
		require.NoError(t, err)
		lines := strings.Split(output, "\n")
		require.Equal(t, 8, countSchedulerEntries(output))

		require.Equal(t, []string{
			"NAMESPACE",
			"NAME",
			"BEGIN",
			"EXPIRATION",
			"SCHEDULE",
			"DUE",
			"TIME",
			"TTL",
			"REPEATS",
			"COUNT",
			"LAST",
			"TRIGGER",
		}, strings.Fields(lines[0]))
	})

	t.Run("yaml", func(t *testing.T) {
		output, err := cmdSchedulerList("-o", "yaml")
		require.NoError(t, err)

		var list []scheduler.ListOutputWide
		require.NoError(t, yaml.Unmarshal([]byte(output), &list))
		assert.Len(t, list, 8)
	})

	t.Run("json", func(t *testing.T) {
		output, err := cmdSchedulerList("-o", "json")
		require.NoError(t, err)

		var list []scheduler.ListOutputWide
		require.NoError(t, json.Unmarshal([]byte(output), &list))
		assert.Len(t, list, 8)
	})

	t.Run("filter", func(t *testing.T) {
		output, err := cmdSchedulerList("-n", "foo")
		require.NoError(t, err)
		assert.Equal(t, 0, countSchedulerEntries(output))

		output, err = cmdSchedulerList("--filter", "all")
		require.NoError(t, err)
		assert.Equal(t, 8, countSchedulerEntries(output))

		output, err = cmdSchedulerList("--filter", "app")
		require.NoError(t, err)
		assert.Equal(t, 2, countSchedulerEntries(output))

		output, err = cmdSchedulerList("--filter", "actor")
		require.NoError(t, err)
		assert.Equal(t, 2, countSchedulerEntries(output))

		output, err = cmdSchedulerList("--filter", "workflow")
		require.NoError(t, err)
		assert.Equal(t, 2, countSchedulerEntries(output))

		output, err = cmdSchedulerList("--filter", "activity")
		require.NoError(t, err)
		assert.Equal(t, 2, countSchedulerEntries(output))
	})
}

func TestSchedulerGet(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping scheduler tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	startDaprRunRetry(t, []int{3510}, func() { cmdStopWithRunTemplate(runFilePath) }, "-f", runFilePath)

	expNames := []string{
		"actor/myactortype/actorid1/test1",
		"actor/myactortype/actorid2/test2",
		"app/test-scheduler/test1",
		"app/test-scheduler/test2",
		"activity/test-scheduler/xyz1::0::1",
		"activity/test-scheduler/xyz2::0::1",
	}

	expWorkflowPrefixes := []string{
		"workflow/test-scheduler/abc1",
		"workflow/test-scheduler/abc2",
	}

	// Wait for all expected items to be present in the scheduler list
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		lines := strings.Split(output, "\n")
		
		// Parse scheduler items
		schedulerItems := make(map[string]bool)
		for _, line := range lines[1:] {
			if strings.TrimSpace(line) == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}
			name := fields[0]
			schedulerItems[name] = true
		}
		
		// Check all expected names are present
		for _, name := range expNames {
			assert.True(c, schedulerItems[name], "expected item %s not found", name)
		}
		
		// Check workflows by prefix
		foundWorkflows := 0
		for name := range schedulerItems {
			for _, prefix := range expWorkflowPrefixes {
				if strings.HasPrefix(name, prefix) {
					foundWorkflows++
					break
				}
			}
		}
		assert.Equal(c, len(expWorkflowPrefixes), foundWorkflows, "expected %d workflow items, found %d", len(expWorkflowPrefixes), foundWorkflows)
	}, 240*time.Second, time.Second)

	t.Run("short", func(t *testing.T) {
		for _, name := range expNames {
			output, err := cmdSchedulerGet(name)
			require.NoError(t, err)
			lines := strings.Split(output, "\n")
			require.Len(t, lines, 3)

			if strings.HasPrefix(name, "activity/") {
				require.Equal(t, []string{
					"NAME",
					"BEGIN",
					"COUNT",
				}, strings.Fields(lines[0]), name)
			} else {
				require.Equal(t, []string{
					"NAME",
					"BEGIN",
					"COUNT",
					"LAST",
					"TRIGGER",
				}, strings.Fields(lines[0]), name)
			}
		}
	})

	t.Run("wide", func(t *testing.T) {
		for _, name := range expNames {
			output, err := cmdSchedulerGet(name, "-o", "wide")
			require.NoError(t, err)
			lines := strings.Split(output, "\n")
			require.Len(t, lines, 3)

			switch {
			case name == "app/test-scheduler/test2":
				require.Equal(t, []string{
					"NAMESPACE",
					"NAME",
					"BEGIN",
					"EXPIRATION",
					"SCHEDULE",
					"DUE",
					"TIME",
					"TTL",
					"REPEATS",
					"COUNT",
					"LAST",
					"TRIGGER",
				}, strings.Fields(lines[0]), name)

			case strings.HasPrefix(name, "activity/"):
				require.Equal(t, []string{
					"NAMESPACE",
					"NAME",
					"BEGIN",
					"DUE",
					"TIME",
					"COUNT",
				}, strings.Fields(lines[0]), name)

			default:
				require.Equal(t, []string{
					"NAMESPACE",
					"NAME",
					"BEGIN",
					"SCHEDULE",
					"DUE",
					"TIME",
					"REPEATS",
					"COUNT",
					"LAST",
					"TRIGGER",
				}, strings.Fields(lines[0]), name)
			}
		}
	})

	t.Run("yaml", func(t *testing.T) {
		for _, name := range expNames {
			output, err := cmdSchedulerGet(name, "-o", "yaml")
			require.NoError(t, err)

			var list []scheduler.ListOutputWide
			require.NoError(t, yaml.Unmarshal([]byte(output), &list))
			assert.Len(t, list, 1)
		}
	})

	t.Run("json", func(t *testing.T) {
		for _, name := range expNames {
			output, err := cmdSchedulerGet(name, "-o", "json")
			require.NoError(t, err)

			var list []scheduler.ListOutputWide
			require.NoError(t, json.Unmarshal([]byte(output), &list))
			assert.Len(t, list, 1)
		}
	})
}

func TestSchedulerDelete(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping scheduler tests in slim mode")
	}

	// Reinstall Dapr to clear any stale scheduler state (workflow entries)
	// from previous tests. Without this, wf.StartWorker hangs because the
	// scheduler container still holds old workflow registrations.
	cmdUninstall()
	ensureDaprInstallation(t)

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	startDaprRunRetry(t, []int{3510}, func() { cmdStopWithRunTemplate(runFilePath) }, "-f", runFilePath)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.GreaterOrEqual(c, countSchedulerEntries(output), 8)
	}, 240*time.Second, time.Second)

	output, err := cmdSchedulerList()
	require.NoError(t, err)

	_, err = cmdSchedulerDelete("actor/myactortype/actorid1/test1")
	require.NoError(t, err)

	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Equal(t, 7, countSchedulerEntries(output))

	_, err = cmdSchedulerDelete(
		"actor/myactortype/actorid2/test2",
		"app/test-scheduler/test1",
		"app/test-scheduler/test2",
	)
	require.NoError(t, err)

	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Equal(t, 4, countSchedulerEntries(output))

	_, err = cmdSchedulerDelete(
		"activity/test-scheduler/xyz1::0::1",
		"activity/test-scheduler/xyz2::0::1",
	)
	require.NoError(t, err)

	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Equal(t, 2, countSchedulerEntries(output))

	lines := strings.Split(output, "\n")
	_, err = cmdSchedulerDelete(
		strings.Fields(lines[1])[0],
		strings.Fields(lines[2])[0],
	)
	require.NoError(t, err)

	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Equal(t, 0, countSchedulerEntries(output))
}

func TestSchedulerDeleteAllAll(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping scheduler tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	startDaprRunRetry(t, []int{3510}, func() { cmdStopWithRunTemplate(runFilePath) }, "-f", runFilePath)

	// On slow macOS CI runners, workflow/activity entries can take over 60s to
	// register, so use a 120s timeout.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.GreaterOrEqual(c, countSchedulerEntries(output), 8)
	}, 240*time.Second, time.Second)

	_, err := cmdSchedulerDeleteAll("all")
	require.NoError(t, err)

	output, err := cmdSchedulerList()
	require.NoError(t, err)
	assert.Equal(t, 0, countSchedulerEntries(output))
}

func TestSchedulerDeleteAll(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping scheduler tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	startDaprRunRetry(t, []int{3510}, func() { cmdStopWithRunTemplate(runFilePath) }, "-f", runFilePath)

	// Wait for all 8 scheduler entries to appear: 2 app jobs, 2 actor
	// reminders, 4 workflow/activity entries. Using countSchedulerEntries
	// avoids hard-coding a line count that breaks if the output format changes.
	// On slow macOS CI runners, workflow/activity entries can take over 60s to
	// register, so use a 120s timeout.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.GreaterOrEqual(c, countSchedulerEntries(output), 8)
	}, 240*time.Second, time.Second)

	_, err := cmdSchedulerDeleteAll("app/test-scheduler")
	require.NoError(t, err)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.Equal(c, 6, countSchedulerEntries(output))
	}, 10*time.Second, 500*time.Millisecond)

	_, err = cmdSchedulerDeleteAll("workflow/test-scheduler/abc1")
	require.NoError(t, err)
	output, err := cmdSchedulerList()
	require.NoError(t, err)
	assert.Equal(t, 5, countSchedulerEntries(output))

	_, err = cmdSchedulerDeleteAll("workflow/test-scheduler")
	require.NoError(t, err)
	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Equal(t, 2, countSchedulerEntries(output))

	_, err = cmdSchedulerDeleteAll("actor/myactortype/actorid1")
	require.NoError(t, err)
	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Equal(t, 1, countSchedulerEntries(output))

	_, err = cmdSchedulerDeleteAll("actor/myactortype")
	require.NoError(t, err)
	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Equal(t, 0, countSchedulerEntries(output))
}

func TestSchedulerExportImport(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping scheduler tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	startDaprRunRetry(t, []int{3510}, func() { cmdStopWithRunTemplate(runFilePath) }, "-f", runFilePath)

	// On slow macOS CI runners, workflow/activity entries can take over 60s to
	// register, so use a 120s timeout.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.GreaterOrEqual(c, countSchedulerEntries(output), 8)
	}, 240*time.Second, time.Second)

	f := filepath.Join(t.TempDir(), "foo")
	_, err := cmdSchedulerExport("-o", f)
	require.NoError(t, err)

	_, err = cmdSchedulerDeleteAll("all")
	require.NoError(t, err)
	output, err := cmdSchedulerList()
	require.NoError(t, err)
	assert.Equal(t, 0, countSchedulerEntries(output))

	_, err = cmdSchedulerImport("-f", f)
	require.NoError(t, err)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.GreaterOrEqual(c, countSchedulerEntries(output), 7)
	}, 60*time.Second, time.Second)
}
