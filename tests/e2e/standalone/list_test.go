//go:build e2e && !template
// +build e2e,!template

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
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestStandaloneList(t *testing.T) {
	ensureDaprInstallation(t)

	executeAgainstRunningDapr(t, func() {
		output, err := cmdList("")
		t.Log(output)
		require.NoError(t, err, "dapr list failed")
		listOutputCheck(t, output, true)

		output, err = cmdList("table")
		t.Log(output)
		require.NoError(t, err, "dapr list failed")
		listOutputCheck(t, output, true)

		output, err = cmdList("json")
		t.Log(output)
		require.NoError(t, err, "dapr list failed")
		listJsonOutputCheck(t, output)

		output, err = cmdList("yaml")
		t.Log(output)
		require.NoError(t, err, "dapr list failed")
		listYamlOutputCheck(t, output)

		output, err = cmdList("invalid")
		t.Log(output)
		require.Error(t, err, "dapr list should fail with an invalid output format")

		// We can call stop so as not to wait for the app to time out
		output, err = cmdStopWithAppID("dapr_e2e_list")
		t.Log(output)
		require.NoError(t, err, "dapr stop failed")
		assert.Contains(t, output, "app stopped successfully: dapr_e2e_list")
	}, "run", "--app-id", "dapr_e2e_list", "-H", "3555", "-G", "4555", "--", "bash", "-c", "sleep 10 ; exit 0")

	t.Run("daprd instance in list", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		path := filepath.Join(homeDir, ".dapr")
		binPath := filepath.Join(path, "bin")
		daprdPath := filepath.Join(binPath, "daprd")

		if runtime.GOOS == "windows" {
			daprdPath += ".exe"
		}

		cmd := exec.Command(daprdPath, "--app-id", "daprd_e2e_list", "--dapr-http-port", "3555", "--dapr-grpc-port", "4555", "--app-port", "0")
		cmd.Start()

		output, err := cmdList("")
		t.Log(output)
		require.NoError(t, err, "dapr list failed with daprd instance")
		listOutputCheck(t, output, false)

		// TODO: remove this condition when `dapr stop` starts working for Windows.
		// See https://github.com/dapr/cli/issues/1034.
		if runtime.GOOS != "windows" {
			output, err = cmdStopWithAppID("daprd_e2e_list")
			t.Log(output)
			require.NoError(t, err, "dapr stop failed")
			assert.Contains(t, output, "app stopped successfully: daprd_e2e_list")
		}

		cmd.Process.Kill()
	})

	t.Run("dashboard instance should not be listed", func(t *testing.T) {
		// TODO: remove this after figuring out the fix.
		// The issue is that the dashboard instance does not gets killed when the app is stopped.
		// This causes issues when uninstalling Dapr, since the .bin folder is not removed on Windows.
		if runtime.GOOS == "windows" {
			t.Skip("skip dashboard test on windows")
		}

		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()

		err := cmdDashboard(ctx, "5555")
		require.NoError(t, err, "dapr dashboard failed")

		output, err := cmdList("")
		t.Log(output)
		require.NoError(t, err, "expected no error status on list")
		require.Equal(t, "No Dapr instances found.\n", output)
	})
}

func listOutputCheck(t *testing.T, output string, isCli bool) {
	lines := strings.Split(output, "\n")[1:] // remove header
	// only one app is runnning at this time
	fields := strings.Fields(lines[0])
	// Fields splits on space, so Created time field might be split again
	assert.GreaterOrEqual(t, len(fields), 4, "expected at least 4 fields in components output")
	if isCli {
		assert.Equal(t, "dapr_e2e_list", fields[0], "expected name to match")
	} else {
		assert.Equal(t, "daprd_e2e_list", fields[0], "expected name to match")
	}
	assert.Equal(t, "3555", fields[1], "expected http port to match")
	assert.Equal(t, "4555", fields[2], "expected grpc port to match")
	assert.Equal(t, "0", fields[3], "expected app port to match")
}

func listJsonOutputCheck(t *testing.T, output string) {
	var result []map[string]interface{}

	err := json.Unmarshal([]byte(output), &result)

	assert.NoError(t, err, "output was not valid JSON")

	assert.Len(t, result, 1, "expected one app to be running")
	assert.Equal(t, "dapr_e2e_list", result[0]["appId"], "expected name to match")
	assert.Equal(t, 3555, int(result[0]["httpPort"].(float64)), "expected http port to match")
	assert.Equal(t, 4555, int(result[0]["grpcPort"].(float64)), "expected grpc port to match")
	assert.Equal(t, 0, int(result[0]["appPort"].(float64)), "expected app port to match")
}

func listYamlOutputCheck(t *testing.T, output string) {
	var result []map[string]interface{}

	err := yaml.Unmarshal([]byte(output), &result)

	assert.NoError(t, err, "output was not valid YAML")

	assert.Len(t, result, 1, "expected one app to be running")
	assert.Equal(t, "dapr_e2e_list", result[0]["appId"], "expected name to match")
	assert.Equal(t, 3555, result[0]["httpPort"], "expected http port to match")
	assert.Equal(t, 4555, result[0]["grpcPort"], "expected grpc port to match")
	assert.Equal(t, 0, result[0]["appPort"], "expected app port to match")
}
