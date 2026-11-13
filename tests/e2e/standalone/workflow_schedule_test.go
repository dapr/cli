//go:build !windows && (e2e || template)

/*
Copyright 2026 The Dapr Authors
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
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/cli/tests/e2e/common"
	"github.com/dapr/cli/tests/e2e/spawn"
)

// schedules parses the JSON output of `dapr workflow schedule get|list`.
func schedules(t require.TestingT, output string) []workflow.ScheduleOutputWide {
	var list []workflow.ScheduleOutputWide
	require.NoError(t, json.Unmarshal([]byte(output), &list), output)
	return list
}

// runIDs lists the instance IDs of the runs a schedule started.
func runIDs(t require.TestingT, appID, name string) []string {
	output, err := cmdWorkflowList(appID, "-o", "json", "--instance-id-prefix", name+"_")
	require.NoError(t, err, output)
	var list []map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &list), output)
	ids := make([]string, 0, len(list))
	for _, item := range list {
		assert.Equal(t, "SimpleWorkflow", item["name"])
		ids = append(ids, item["instanceID"].(string))
	}
	return ids
}

func TestWorkflowSchedule(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping workflow tests in slim mode")
	}
	if v := common.GetRuntimeVersion(t, false); v.LessThan(common.VersionWithWorkflowSchedules) {
		t.Skipf("workflow schedules need runtime %s, testing %s", common.VersionWithWorkflowSchedules, v)
	}

	runFilePath := "../testdata/run-template-files/test-workflow.yaml"
	appID := "test-workflow"
	startDaprRun(t, []int{3510}, func() { cmdStopWithAppID(appID) }, "-f", runFilePath)

	waitForAppHealthy(t, 60*time.Second, appID)
	purgeOut, purgeErr := cmdWorkflowPurge(appID, redisConnString, "--all")
	require.NoError(t, purgeErr, purgeOut)

	// Later tests expect no workflow instance to be left behind.
	t.Cleanup(func() {
		for _, name := range []string{"every-second", "manual"} {
			_, _ = cmdWorkflowSchedule(appID, "rm", name)
		}
		// purge --all skips instances still running, so retry until the runs finish.
		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			out, err := cmdWorkflowPurge(appID, redisConnString, "--all")
			require.NoError(c, err, out)
			out, err = cmdWorkflowList(appID)
			require.NoError(c, err, out)
			assert.Contains(c, out, "No workflow found")
		}, 30*time.Second, time.Second)
	})

	t.Run("list empty", func(t *testing.T) {
		output, err := cmdWorkflowSchedule(appID, "list")
		require.NoError(t, err, output)
		assert.Equal(t, `❌  No workflow schedule found in namespace "default" for app ID "test-workflow"
`, output)
	})

	t.Run("missing", func(t *testing.T) {
		for _, sub := range []string{"get", "rm", "run"} {
			output, err := cmdWorkflowSchedule(appID, sub, "nope")
			require.Error(t, err, sub)
			assert.Contains(t, output, "not found", sub)
		}
	})

	t.Run("invalid create", func(t *testing.T) {
		output, err := cmdWorkflowSchedule(appID, "create", "bad")
		require.Error(t, err, output)
		assert.Contains(t, output, "required flag(s)")

		_, err = cmdWorkflowSchedule(appID, "create", "bad", "-w", "SimpleWorkflow", "-s", "* * *")
		require.Error(t, err)

		_, err = cmdWorkflowSchedule(appID, "create", "dapr-internal-schedule-x", "-w", "SimpleWorkflow", "-s", "@every 1s")
		require.Error(t, err)
	})

	t.Run("create", func(t *testing.T) {
		output, err := cmdWorkflowSchedule(appID, "add", "every-second",
			"-w", "SimpleWorkflow", "-s", "@every 1s", "-x", `{"k":"v"}`, "--repeats", "2")
		require.NoError(t, err, output)
		assert.Contains(t, output, `Workflow schedule "every-second" created`)

		output, err = cmdWorkflowSchedule(appID, "create", "every-second", "-w", "SimpleWorkflow", "-s", "@every 1s")
		require.Error(t, err, output)
		assert.Contains(t, output, "already exists")
	})

	t.Run("read", func(t *testing.T) {
		output, err := cmdWorkflowSchedule(appID, "ls", "-o", "json")
		require.NoError(t, err, output)
		list := schedules(t, output)
		require.Len(t, list, 1)
		assert.Equal(t, "every-second", list[0].Name)
		assert.Equal(t, "SimpleWorkflow", list[0].Workflow)
		assert.Equal(t, "@every 1s", list[0].Schedule)
		assert.Equal(t, `SimpleWorkflow({"k":"v"})`, list[0].Runs)
		// Both runs can be over already on a slow runner; "runs complete" checks the end state.
		assert.Contains(t, []string{"active", "completed"}, list[0].State)

		output, err = cmdWorkflowSchedule(appID, "show", "every-second", "-o", "json")
		require.NoError(t, err, output)
		list = schedules(t, output)
		require.Len(t, list, 1)
		assert.Equal(t, "every-second", list[0].Name)

		output, err = cmdWorkflowSchedule(appID, "ls")
		require.NoError(t, err, output)
		assert.Equal(t, []string{"NAME", "NEXT", "LAST", "STATUS"}, strings.Fields(strings.Split(output, "\n")[0]))

		output, err = cmdWorkflowSchedule(appID, "ls", "-o", "wide")
		require.NoError(t, err, output)
		// The table leaves out a column blank on every row, as RESULT is until a run has finished.
		header := slices.DeleteFunc(strings.Fields(strings.Split(output, "\n")[0]), func(h string) bool { return h == "RESULT" })
		assert.Equal(t, []string{"NAME", "SCHEDULE", "RUNS", "NEXT", "LAST", "STATUS"}, header)

		output, err = cmdWorkflowSchedule(appID, "get", "every-second", "-o", "yaml")
		require.NoError(t, err, output)
		var fromYAML []workflow.ScheduleOutputWide
		require.NoError(t, yaml.Unmarshal([]byte(output), &fromYAML), output)
		assert.Len(t, fromYAML, 1)

		for _, alias := range [][]string{
			{"wf", "sched", "ls"},
			{"workflow", "schedules", "list"},
		} {
			output, err = spawn.Command(common.GetDaprPath(), append(alias, "-a", appID, "-o", "json")...)
			require.NoError(t, err, output)
			list = schedules(t, output)
			require.Len(t, list, 1, alias)
			assert.Equal(t, "every-second", list[0].Name, alias)
		}
	})

	t.Run("runs complete", func(t *testing.T) {
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			output, err := cmdWorkflowSchedule(appID, "get", "every-second", "-o", "json")
			require.NoError(c, err, output)
			list := schedules(c, output)
			require.Len(c, list, 1)
			assert.Equal(c, "completed", list[0].State)
			assert.Equal(c, uint32(2), list[0].RunCount)
			assert.True(c, strings.HasPrefix(list[0].Status, "ok "), list[0].Status)
			assert.Contains(c, list[0].Result, "Processed simple activity")
			assert.True(c, strings.HasPrefix(list[0].LastInstanceID, "every-second_"), list[0].LastInstanceID)
		}, 60*time.Second, time.Second)

		assert.Len(t, runIDs(t, appID, "every-second"), 2)

		output, err := cmdWorkflowSchedule(appID, "run", "every-second")
		require.Error(t, err, output)
		assert.Contains(t, output, "no runs left")
	})

	t.Run("trigger", func(t *testing.T) {
		output, err := cmdWorkflowSchedule(appID, "create", "manual", "-w", "SimpleWorkflow", "-s", "0 0 0 1 1 *")
		require.NoError(t, err, output)

		output, err = cmdWorkflowSchedule(appID, "run", "manual")
		require.NoError(t, err, output)
		assert.Contains(t, output, `Workflow schedule "manual" triggered, starting run: manual_`)
		_, id, _ := strings.Cut(output, "starting run: ")
		id = strings.TrimSpace(id)
		assert.True(t, strings.HasSuffix(id, "_manual"), id)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			out, gerr := cmdWorkflowSchedule(appID, "get", "manual", "-o", "json")
			require.NoError(c, gerr, out)
			list := schedules(c, out)
			require.Len(c, list, 1)
			assert.Equal(c, uint32(1), list[0].RunCount)
			assert.Equal(c, id, list[0].LastInstanceID)
			assert.Contains(c, runIDs(c, appID, "manual"), id)
		}, 30*time.Second, time.Second)

		output, err = cmdWorkflowSchedule(appID, "create", "manual", "-w", "SimpleWorkflow", "-s", "0 0 0 1 1 *", "--overwrite")
		require.NoError(t, err, output)
		output, err = cmdWorkflowSchedule(appID, "get", "manual", "-o", "json")
		require.NoError(t, err, output)
		assert.Equal(t, uint32(0), schedules(t, output)[0].RunCount)
	})

	t.Run("delete", func(t *testing.T) {
		output, err := cmdWorkflowSchedule(appID, "rm", "manual")
		require.NoError(t, err, output)
		assert.Contains(t, output, `Workflow schedule "manual" deleted`)

		for _, sub := range []string{"get", "rm"} {
			output, err = cmdWorkflowSchedule(appID, sub, "manual")
			require.Error(t, err, sub)
			assert.Contains(t, output, "not found", sub)
		}

		output, err = cmdWorkflowSchedule(appID, "ls", "-o", "json")
		require.NoError(t, err, output)
		list := schedules(t, output)
		require.Len(t, list, 1)
		assert.Equal(t, "every-second", list[0].Name)

		assert.Len(t, runIDs(t, appID, "manual"), 1, "deleting a schedule leaves its runs")
	})
}
