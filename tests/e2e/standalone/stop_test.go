//go:build e2e && !template

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
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandaloneStop(t *testing.T) {
	ensureDaprInstallation(t)

	time.Sleep(5 * time.Second)

	t.Cleanup(func() {
		// remove dapr installation after all tests in this function.
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	runArgs := []string{"run", "--app-id", "dapr_e2e_stop", "--"}
	if runtime.GOOS == "windows" {
		runArgs = append(runArgs, "cmd", "/c", "ping -n 61 127.0.0.1 >nul")
	} else {
		runArgs = append(runArgs, "bash", "-c", "sleep 60 ; exit 1")
	}
	executeAgainstRunningDapr(t, func() {
		t.Run("stop", func(t *testing.T) {
			output, err := cmdStopWithAppID("dapr_e2e_stop")
			t.Log(output)
			require.NoError(t, err, "dapr stop failed")
			assert.Contains(t, output, "app stopped successfully: dapr_e2e_stop")
		})
	}, runArgs...)

	t.Run("stop with unknown flag", func(t *testing.T) {
		output, err := cmdStopWithAppID("dapr_e2e_stop", "-p", "test")
		require.Error(t, err, "expected error on stop with unknown flag")
		require.Contains(t, output, "Error: unknown shorthand flag: 'p' in -p\nUsage:", "expected usage to be printed")
		require.Contains(t, output, "-a, --app-id string", "expected usage to be printed")
		require.Contains(t, output, "-f, --run-file string", "expected usage to be printed")
	})
}
