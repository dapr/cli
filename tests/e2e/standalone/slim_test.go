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
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapr/cli/tests/e2e/common"
)

// TestSlimRunStartsLocalServices runs a workflow on a slim installation.
// The run file sets no placement or scheduler address, so `dapr run` starts
// both services from the slim installation and stops them when it exits.
func TestSlimRunStartsLocalServices(t *testing.T) {
	if !isSlimMode() {
		t.Skip("dapr run starts local services for slim installations only")
	}
	if !common.GetRuntimeVersion(t, false).GreaterThanEqual(common.VersionWithScheduler) {
		t.Skip("the runtime has no scheduler")
	}

	waitForPortsFree(t, 50005, 50006)

	runFilePath := "../testdata/run-template-files/test-workflow-slim.yaml"
	appID := "test-workflow-slim"
	startDaprRun(t, []int{3511}, func() { cmdStopWithAppID(appID) }, "-f", runFilePath)

	waitForAppHealthy(t, 60*time.Second, appID)

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	daprPath := filepath.Join(homeDir, ".dapr")

	t.Run("dapr run started the scheduler and placement services", func(t *testing.T) {
		for _, port := range []string{"50005", "50006"} {
			conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), time.Second)
			require.NoError(t, err, "nothing listens on port %s", port)
			conn.Close()
		}
		assert.FileExists(t, filepath.Join(daprPath, "logs", "scheduler.log"))
		assert.FileExists(t, filepath.Join(daprPath, "logs", "placement.log"))
	})

	t.Run("a workflow completes", func(t *testing.T) {
		output, err := cmdWorkflowRun(appID, "SimpleWorkflow", "--instance-id=slim-1", "--input", `"slim"`)
		require.NoError(t, err, output)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			output, err := cmdWorkflowList(appID, "-o", "json")
			if !assert.NoError(c, err, output) {
				return
			}
			var list []map[string]interface{}
			if !assert.NoError(c, json.Unmarshal([]byte(output), &list)) {
				return
			}
			status := ""
			for _, item := range list {
				if item["instanceID"] == "slim-1" {
					status, _ = item["runtimeStatus"].(string)
				}
			}
			assert.Equal(c, "COMPLETED", status)
		}, 60*time.Second, time.Second)
	})

	t.Run("the services stop with dapr run", func(t *testing.T) {
		_, err := cmdStopWithAppID(appID)
		require.NoError(t, err)
		waitForPortsFree(t, 50005, 50006)
	})
}
