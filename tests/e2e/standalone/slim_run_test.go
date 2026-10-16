//go:build e2e && !template

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
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapr/cli/tests/e2e/common"
)

// TestSlimSingleAppRunStartsLocalServices runs one app with `dapr run` on a
// slim installation, also on Windows. The command sets no placement or
// scheduler address, so `dapr run` starts the services of the slim
// installation. When the app exits, `dapr run` stops daprd and then the
// services.
func TestSlimSingleAppRunStartsLocalServices(t *testing.T) {
	if !isSlimMode() {
		t.Skip("dapr run starts local services for slim installations only")
	}
	if !common.GetRuntimeVersion(t, false).GreaterThanEqual(common.VersionWithScheduler) {
		t.Skip("the runtime has no scheduler")
	}

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	binDir := filepath.Join(homeDir, ".dapr", "bin")

	ext := ""
	placementPort, schedulerPort := 50005, 50006
	if runtime.GOOS == "windows" {
		ext = ".exe"
		placementPort, schedulerPort = 6050, 6060
	}
	// `dapr init --slim --scheduler-placement` installs no placement binary.
	_, statErr := os.Stat(filepath.Join(binDir, "placement"+ext))
	placementInstalled := statErr == nil
	services := map[string]int{"scheduler": schedulerPort}
	if placementInstalled {
		services["placement"] = placementPort
	}
	ports := make([]int, 0, len(services))
	for _, port := range services {
		ports = append(ports, port)
	}

	waitForPortsFree(t, ports...)

	// The app runs long enough for the checks below, then it exits.
	args := []string{"--app-id", "slim-single-app", "--"}
	if runtime.GOOS == "windows" {
		args = append(args, "cmd", "/c", "ping -n 21 127.0.0.1 > nul")
	} else {
		args = append(args, "bash", "-c", "sleep 20")
	}

	var output string
	var runErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		output, runErr = cmdRun("", args...)
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		case <-time.After(2 * time.Minute):
		}
	})

	t.Run("dapr run started the services", func(t *testing.T) {
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			for name, port := range services {
				conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
				if assert.NoError(c, err, "the %s service does not listen on port %d", name, port) {
					conn.Close()
				}
			}
		}, 30*time.Second, 500*time.Millisecond)
		for name := range services {
			assert.FileExists(t, filepath.Join(homeDir, ".dapr", "logs", name+".log"))
		}
	})

	t.Run("the services stop when the app exits", func(t *testing.T) {
		select {
		case <-done:
		case <-time.After(2 * time.Minute):
			require.FailNow(t, "dapr run did not exit")
		}
		require.NoError(t, runErr, output)
		assert.Contains(t, output, "Exited Dapr successfully")
		for name := range services {
			assert.Contains(t, output, "Started the "+name+" service")
			assert.Contains(t, output, "Stopped the "+name+" service")
		}
		waitForPortsFree(t, ports...)
	})
}

// TestSlimSingleAppRunWithoutLocalServices runs one app with
// --start-local-services=false on a slim installation, also on Windows.
// The command sets no placement or scheduler address, but `dapr run` starts
// no service, and the ports stay free.
func TestSlimSingleAppRunWithoutLocalServices(t *testing.T) {
	if !isSlimMode() {
		t.Skip("dapr run starts local services for slim installations only")
	}
	if !common.GetRuntimeVersion(t, false).GreaterThanEqual(common.VersionWithScheduler) {
		t.Skip("the runtime has no scheduler")
	}

	ports := []int{50005, 50006}
	if runtime.GOOS == "windows" {
		ports = []int{6050, 6060}
	}
	waitForPortsFree(t, ports...)

	// The app runs long enough for the checks below, then it exits.
	appID := "slim-no-local-services"
	args := []string{"--app-id", appID, "--start-local-services=false", "--"}
	if runtime.GOOS == "windows" {
		args = append(args, "cmd", "/c", "ping -n 16 127.0.0.1 > nul")
	} else {
		args = append(args, "bash", "-c", "sleep 15")
	}

	var output string
	var runErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		output, runErr = cmdRun("", args...)
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		case <-time.After(2 * time.Minute):
		}
	})

	t.Run("no service listens while the app runs", func(t *testing.T) {
		waitForAppsListed(t, 60*time.Second, appID)
		for _, port := range ports {
			conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
			if err == nil {
				conn.Close()
			}
			assert.Error(t, err, "a service listens on port %d", port)
		}
	})

	t.Run("dapr run started no service", func(t *testing.T) {
		select {
		case <-done:
		case <-time.After(2 * time.Minute):
			require.FailNow(t, "dapr run did not exit")
		}
		require.NoError(t, runErr, output)
		assert.Contains(t, output, "No scheduler service answers on")
		assert.NotContains(t, output, "Started the placement service")
		assert.NotContains(t, output, "Started the scheduler service")
		waitForPortsFree(t, ports...)
	})
}
