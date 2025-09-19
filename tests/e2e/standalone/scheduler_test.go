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
	cleanUpLogs()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		// remove dapr installation after all tests in this function.
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/jobs.yaml"
	t.Cleanup(func() {
		// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
		cleanUpLogs()
		waitAppsToBeStopped()
	})
	args := []string{
		"-f", runFilePath,
	}

	go cmdRunWithContext(t.Context(), "", args...)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.Len(c, strings.Lines(output), 5)
	}, time.Second*10, time.Millisecond*10)

	t.Run("short", func(t *testing.T) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		lines := strings.Split(output, "\n")
		require.Len(t, lines, 5)

		require.Equal(t, []string{
			"NAMESPACE",
			"APP ID",
			"NAME",
			"TARGET",
			"BEGIN",
			"COUNT",
			"LAST",
			"TRIGGER",
		}, strings.Fields(lines[0]))

		expNames := []string{"test1", "test2", "test1", "test2"}
		expTargets := []string{"jobs", "jobs", "myactortype||actorid1", "myactortype||actorid2"}
		for i, line := range lines[1:] {
			assert.Equal(t, "default", strings.Fields(line)[0])

			assert.Equal(t, "jobs", strings.Fields(line)[1])

			assert.Equal(t, expNames[i], strings.Fields(line)[2])

			assert.Equal(t, expTargets[i], strings.Fields(line)[3])

			assert.NotEmpty(t, strings.Fields(line)[4])

			count, err := strconv.Atoi(strings.Fields(line)[5])
			require.NoError(t, err)
			assert.Equal(t, 1, count)

			assert.NotEmpty(t, strings.Fields(line)[6])
		}
	})

	t.Run("wide", func(t *testing.T) {
		output, err := cmdSchedulerList("-o", "wide")
		require.NoError(t, err)
		lines := strings.Split(output, "\n")
		require.Len(t, lines, 5)

		require.Equal(t, []string{
			"NAMESPACE",
			"APP ID",
			"NAME",
			"TARGET",
			"BEGIN",
			"EXPIRATION",
			"SCHEDULE",
			"DUE TIME",
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
		assert.Len(t, list, 4)
	})

	t.Run("json", func(t *testing.T) {
		output, err := cmdSchedulerList("-o", "json")
		require.NoError(t, err)

		var list []scheduler.ListOutputWide
		require.NoError(t, json.Unmarshal([]byte(output), &list))
		assert.Len(t, list, 4)
	})

	t.Run("filter", func(t *testing.T) {
		output, err := cmdSchedulerList("-n", "foo")
		require.NoError(t, err)
		assert.Len(t, strings.Lines(output), 1)

		output, err = cmdSchedulerList("--filter-type", "all")
		require.NoError(t, err)
		assert.Len(t, strings.Lines(output), 5)

		output, err = cmdSchedulerList("--filter-type", "jobs")
		require.NoError(t, err)
		assert.Len(t, strings.Lines(output), 3)

		output, err = cmdSchedulerList("--filter-type", "actorreminder")
		require.NoError(t, err)
		assert.Len(t, strings.Lines(output), 3)
	})
}

func TestSchedulerDelete(t *testing.T) {
	cleanUpLogs()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		// remove dapr installation after all tests in this function.
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/jobs.yaml"
	t.Cleanup(func() {
		// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
		cleanUpLogs()
		waitAppsToBeStopped()
	})
	args := []string{
		"-f", runFilePath,
	}

	go cmdRunWithContext(t.Context(), "", args...)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.Len(c, strings.Lines(output), 5)
	}, time.Second*10, time.Millisecond*10)

	// No namespace
	_, err := cmdSchedulerDelete("job/jobs/test1")
	require.NoError(t, err)

	var output string
	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Lines(output), 5)

	_, err = cmdSchedulerDelete("job/jobs/test1", "-n", "default")
	require.NoError(t, err)
	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Lines(output), 4)

	_, err = cmdSchedulerDelete("-n", "default", "actorreminder/myactortype||actor1||test1")
	require.NoError(t, err)
	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Lines(output), 3)

	_, err = cmdSchedulerDelete("-n", "default",
		"job/jobs/test2",
		"actorreminder/myactortype||actor2||test2",
	)
	require.NoError(t, err)
	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Lines(output), 1)
}

func TestSchedulerDeleteAll(t *testing.T) {
	cleanUpLogs()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		// remove dapr installation after all tests in this function.
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/jobs.yaml"
	t.Cleanup(func() {
		// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
		cleanUpLogs()
		waitAppsToBeStopped()
	})
	args := []string{
		"-f", runFilePath,
	}

	go cmdRunWithContext(t.Context(), "", args...)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.Len(c, strings.Lines(output), 5)
	}, time.Second*10, time.Millisecond*10)

	// No namespace
	_, err := cmdSchedulerDelete("job/jobs/test1")
	require.NoError(t, err)

	var output string
	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Lines(output), 5)

	_, err = cmdSchedulerDelete("--delete-all-yes-i-know-what-i-am-doing")
	require.NoError(t, err)
	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Lines(output), 1)
}

func TestSchedulerExportImport(t *testing.T) {
	cleanUpLogs()
	ensureDaprInstallation(t)
	t.Cleanup(func() {
		// remove dapr installation after all tests in this function.
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runFilePath := "../testdata/run-template-files/jobs.yaml"
	t.Cleanup(func() {
		// assumption in the test is that there is only one set of app and daprd logs in the logs directory.
		cleanUpLogs()
		waitAppsToBeStopped()
	})
	args := []string{
		"-f", runFilePath,
	}

	go cmdRunWithContext(t.Context(), "", args...)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		output, err := cmdSchedulerList()
		require.NoError(t, err)
		assert.Len(c, strings.Lines(output), 5)
	}, time.Second*10, time.Millisecond*10)

	f := filepath.Join(t.TempDir(), "foo")
	_, err = cmdSchedulerExport("-o", f)
	require.NoError(t, err)

	_, err = cmdSchedulerDelete("--delete-all-yes-i-know-what-i-am-doing")
	require.NoError(t, err)
	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Lines(output), 1)

	_, err = cmdSchedulerImport("-f", f)
	require.NoError(t, err)
	output, err = cmdSchedulerList()
	require.NoError(t, err)
	assert.Len(t, strings.Lines(output), 5)
}
