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

func TestSchedulerList(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping scheduler tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	t.Cleanup(func() {
		cmdStopWithRunTemplate(runFilePath)
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	args := []string{"-f", runFilePath}
	go func() {
		o, err := cmdRun("", args...)
		t.Log(o)
		t.Log(err)
	}()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.Len(c, strings.Split(output, "\n"), 10)
	}, time.Second*30, time.Millisecond*10)

	time.Sleep(time.Second * 3)

	t.Run("short", func(t *testing.T) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		lines := strings.Split(output, "\n")
		require.Len(t, lines, 10)

		require.Equal(t, []string{
			"NAME",
			"BEGIN",
			"COUNT",
			"LAST",
			"TRIGGER",
		}, strings.Fields(lines[0]))

		expNames := []string{
			"actor/myactortype/actorid1/test1",
			"actor/myactortype/actorid2/test2",
			"app/test-scheduler/test1",
			"app/test-scheduler/test2",
		}
		for i, line := range lines[1:5] {
			assert.Equal(t, expNames[i], strings.Fields(line)[0])

			assert.NotEmpty(t, strings.Fields(line)[1])

			count, err := strconv.Atoi(strings.Fields(line)[2])
			require.NoError(t, err)
			assert.Equal(t, 1, count)
		}

		expNames = []string{
			"activity/test-scheduler/xyz1::0::1",
			"activity/test-scheduler/xyz2::0::1",
		}
		for i, line := range lines[5:7] {
			assert.Equal(t, expNames[i], strings.Fields(line)[0])

			assert.NotEmpty(t, strings.Fields(line)[1])

			count, err := strconv.Atoi(strings.Fields(line)[2])
			require.NoError(t, err)
			assert.Equal(t, 0, count)
			if err != nil {
				return
			}
		}

		expNames = []string{
			"workflow/test-scheduler/abc1",
			"workflow/test-scheduler/abc2",
		}
		for i, line := range lines[7:9] {
			assert.True(t, strings.HasPrefix(strings.Fields(line)[0], expNames[i]), strings.Fields(line)[0])
		}
	})

	t.Run("wide", func(t *testing.T) {
		output, err := cmdSchedulerList("-o", "wide")
		require.NoError(t, err)
		lines := strings.Split(output, "\n")
		require.Len(t, lines, 10)

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
		assert.Len(t, strings.Split(output, "\n"), 2)

		output, err = cmdSchedulerList("--filter", "all")
		require.NoError(t, err)
		assert.Len(t, strings.Split(output, "\n"), 10)

		output, err = cmdSchedulerList("--filter", "app")
		require.NoError(t, err)
		assert.Len(t, strings.Split(output, "\n"), 4)

		output, err = cmdSchedulerList("--filter", "actor")
		require.NoError(t, err)
		assert.Len(t, strings.Split(output, "\n"), 4)

		output, err = cmdSchedulerList("--filter", "workflow")
		require.NoError(t, err)
		assert.Len(t, strings.Split(output, "\n"), 4)

		output, err = cmdSchedulerList("--filter", "activity")
		require.NoError(t, err)
		assert.Len(t, strings.Split(output, "\n"), 4)
	})
}

func TestSchedulerGet(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping scheduler tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	t.Cleanup(func() {
		cmdStopWithRunTemplate(runFilePath)
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	args := []string{"-f", runFilePath}

	go func() {
		o, err := cmdRun("", args...)
		t.Log(o)
		t.Log(err)
	}()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.Len(c, strings.Split(output, "\n"), 10)
	}, time.Second*30, time.Millisecond*10)

	expNames := []string{
		"actor/myactortype/actorid1/test1",
		"actor/myactortype/actorid2/test2",
		"app/test-scheduler/test1",
		"app/test-scheduler/test2",
		"activity/test-scheduler/xyz1::0::1",
		"activity/test-scheduler/xyz2::0::1",
	}

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

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	t.Cleanup(func() {
		cmdStopWithRunTemplate(runFilePath)
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})
	args := []string{"-f", runFilePath}

	go func() {
		for range 10 {
			o, err := cmdRun("", args...)
			t.Log(o)
			t.Log(err)
			if err == nil {
				break
			}
			time.Sleep(time.Second * 2)
		}
	}()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.Len(c, strings.Split(output, "\n"), 10)
	}, time.Second*30, time.Millisecond*10)

	output, err := cmdSchedulerList()
	require.NoError(t, err)

	_, err = cmdSchedulerDelete("actor/myactortype/actorid1/test1")
	require.NoError(t, err)

	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Split(output, "\n"), 9)

	_, err = cmdSchedulerDelete(
		"actor/myactortype/actorid2/test2",
		"app/test-scheduler/test1",
		"app/test-scheduler/test2",
	)
	require.NoError(t, err)

	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Split(output, "\n"), 6)

	_, err = cmdSchedulerDelete(
		"activity/test-scheduler/xyz1::0::1",
		"activity/test-scheduler/xyz2::0::1",
	)
	require.NoError(t, err)

	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Split(output, "\n"), 4)

	_, err = cmdSchedulerDelete(
		strings.Fields(strings.Split(output, "\n")[1])[0],
		strings.Fields(strings.Split(output, "\n")[2])[0],
	)
	require.NoError(t, err)

	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Split(output, "\n"), 2)
}

func TestSchedulerDeleteAllAll(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping scheduler tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	t.Cleanup(func() {
		cmdStopWithRunTemplate(runFilePath)
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})
	args := []string{"-f", runFilePath}

	go func() {
		for range 10 {
			o, err := cmdRun("", args...)
			t.Log(o)
			t.Log(err)
			if err == nil {
				break
			}
			time.Sleep(time.Second * 2)
		}
	}()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.Len(c, strings.Split(output, "\n"), 10)
	}, time.Second*30, time.Millisecond*10)

	_, err := cmdSchedulerDeleteAll("all")
	require.NoError(t, err)

	output, err := cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Split(output, "\n"), 2)
}

func TestSchedulerDeleteAll(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping scheduler tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	t.Cleanup(func() {
		cmdStopWithRunTemplate(runFilePath)
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})
	args := []string{"-f", runFilePath}

	go func() {
		for range 10 {
			o, err := cmdRun("", args...)
			t.Log(o)
			t.Log(err)
			if err == nil {
				break
			}
			time.Sleep(time.Second * 2)
		}
	}()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.GreaterOrEqual(c, len(strings.Split(output, "\n")), 7)
	}, time.Second*30, time.Millisecond*10)

	_, err := cmdSchedulerDeleteAll("app/test-scheduler")
	require.NoError(t, err)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList("--filter", "app")
		require.NoError(t, err)
		assert.Len(c, strings.Split(output, "\n"), 2)
	}, time.Second*30, time.Millisecond*10)

	_, err = cmdSchedulerDeleteAll("workflow/test-scheduler/abc1")
	require.NoError(t, err)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList("--filter", "workflow")
		require.NoError(t, err)
		assert.NotContains(c, output, "workflow/test-scheduler/abc1")
	}, time.Second*30, time.Millisecond*10)

	_, err = cmdSchedulerDeleteAll("workflow/test-scheduler")
	require.NoError(t, err)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList("--filter", "workflow")
		require.NoError(t, err)
		assert.Len(c, strings.Split(output, "\n"), 2)
	}, time.Second*30, time.Millisecond*10)

	_, err = cmdSchedulerDeleteAll("actor/myactortype/actorid1")
	require.NoError(t, err)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList("--filter", "actor")
		require.NoError(t, err)
		assert.NotContains(c, output, "actor/myactortype/actorid1")
	}, time.Second*30, time.Millisecond*10)

	_, err = cmdSchedulerDeleteAll("actor/myactortype")
	require.NoError(t, err)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList("--filter", "actor")
		require.NoError(t, err)
		assert.Len(c, strings.Split(output, "\n"), 2)
	}, time.Second*30, time.Millisecond*10)
}

func TestSchedulerExportImport(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping scheduler tests in slim mode")
	}

	cmdUninstall()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/test-scheduler.yaml"
	t.Cleanup(func() {
		cmdStopWithRunTemplate(runFilePath)
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})
	args := []string{"-f", runFilePath}

	go func() {
		for range 10 {
			o, err := cmdRun("", args...)
			t.Log(o)
			t.Log(err)
			if err == nil {
				break
			}
			time.Sleep(time.Second * 2)
		}
	}()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.Len(c, strings.Split(output, "\n"), 10)
	}, time.Second*30, time.Millisecond*10)

	f := filepath.Join(t.TempDir(), "foo")
	_, err := cmdSchedulerExport("-o", f)
	require.NoError(t, err)

	_, err = cmdSchedulerDeleteAll("all")
	require.NoError(t, err)
	output, err := cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Split(output, "\n"), 2)

	_, err = cmdSchedulerImport("-f", f)
	require.NoError(t, err)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.GreaterOrEqual(c, len(strings.Split(output, "\n")), 9)
	}, time.Second*30, time.Millisecond*10)
}
