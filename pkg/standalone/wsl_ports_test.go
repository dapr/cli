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

package standalone

import (
	"errors"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDockerDesktop models Docker Desktop on the WSL2 backend: its engine
// runs inside the docker-desktop WSL distro, so `wsl --shutdown` kills it
// until Docker Desktop restarts the distro.
type fakeDockerDesktop struct {
	engineUp bool
	winNATUp bool
	calls    []string
}

var errEnginePing = errors.New("request returned 500 Internal Server Error for API route and version " +
	"http://%2F%2F.%2Fpipe%2FdockerDesktopLinuxEngine/_ping, check if the server supports the requested API version")

// ops returns wslPortOps wired to the fake. portErr is what the scheduler
// port check reports; winNATRunning is the initial WinNAT state.
func (f *fakeDockerDesktop) ops(portErr error, winNATRunning bool) wslPortOps {
	f.winNATUp = winNATRunning
	return wslPortOps{
		elevated:   func() bool { return true },
		checkPorts: func() error { f.calls = append(f.calls, "checkPorts"); return portErr },
		shutdownWSL: func() error {
			f.calls = append(f.calls, "shutdownWSL")
			f.engineUp = false
			return nil
		},
		isWinNATRunning: func() bool { return f.winNATUp },
		stopWinNAT:      func() error { f.calls = append(f.calls, "stopWinNAT"); f.winNATUp = false; return nil },
		startWinNAT:     func() error { f.calls = append(f.calls, "startWinNAT"); f.winNATUp = true; return nil },
		startWSL:        func() { f.calls = append(f.calls, "startWSL") },
		waitForRuntime: func() error {
			// Docker Desktop auto-restarts its distro after wsl --shutdown.
			f.calls = append(f.calls, "waitForRuntime")
			f.engineUp = true
			return nil
		},
	}
}

// dockerRun fails the way the Docker CLI does when the engine is down.
func (f *fakeDockerDesktop) dockerRun() error {
	f.calls = append(f.calls, "dockerRun")
	if !f.engineUp {
		return errEnginePing
	}
	return nil
}

// initContainers models Init's parallel container steps (placement,
// scheduler, redis, zipkin): all of them need the engine to be up.
func (f *fakeDockerDesktop) initContainers() error {
	for range []string{"placement", "scheduler", "redis", "zipkin"} {
		if err := f.dockerRun(); err != nil {
			return err
		}
	}
	return nil
}

// TestPrepareSchedulerHostPorts_DockerDesktopRepro reproduces the report where
// an elevated `dapr init` on Windows with Docker Desktop (WSL2 backend)
// failed with "500 Internal Server Error ... dockerDesktopLinuxEngine/_ping".
// The CLI ran `wsl --shutdown` unconditionally, killing the Docker engine,
// and then ran `docker run` straight into the dead engine.
func TestPrepareSchedulerHostPorts_DockerDesktopRepro(t *testing.T) {
	t.Run("ports free: WSL is left alone and all containers start", func(t *testing.T) {
		f := &fakeDockerDesktop{engineUp: true}

		restore, err := prepareSchedulerHostPorts(f.ops(nil, true))
		require.NoError(t, err)
		assert.Nil(t, restore, "nothing was stopped, so there is nothing to restore")
		require.NoError(t, f.initContainers(), "docker run must not hit a dead engine when no port conflict exists")

		assert.Equal(t, []string{"checkPorts", "dockerRun", "dockerRun", "dockerRun", "dockerRun"}, f.calls)
	})

	t.Run("port busy: engine is back before any container starts", func(t *testing.T) {
		f := &fakeDockerDesktop{engineUp: true}
		restore, err := prepareSchedulerHostPorts(f.ops(errors.New("port 2379 is not available"), true))
		require.NoError(t, err)
		require.NotNil(t, restore)
		require.NoError(t, f.initContainers(), "no container may start before Docker Desktop restarts its engine")
		restore()

		assert.Equal(t, []string{
			"checkPorts", "shutdownWSL", "stopWinNAT", "waitForRuntime",
			"dockerRun", "dockerRun", "dockerRun", "dockerRun",
			"startWinNAT", "startWSL",
		}, f.calls)
	})
}

func TestPrepareSchedulerHostPorts(t *testing.T) {
	portBusy := errors.New("port 2379 is not available")

	t.Run("non-elevated with busy port returns guidance and touches nothing", func(t *testing.T) {
		f := &fakeDockerDesktop{engineUp: true}
		ops := f.ops(portBusy, true)
		ops.elevated = func() bool { return false }

		restore, err := prepareSchedulerHostPorts(ops)
		require.Error(t, err)
		assert.Nil(t, restore)
		assert.Contains(t, err.Error(), "port 2379 is not available")
		assert.Contains(t, err.Error(), "elevated (Administrator)")
		assert.Equal(t, []string{"checkPorts"}, f.calls)
	})

	t.Run("WinNAT not running is neither stopped nor started", func(t *testing.T) {
		f := &fakeDockerDesktop{engineUp: true}
		restore, err := prepareSchedulerHostPorts(f.ops(portBusy, false))
		require.NoError(t, err)
		require.NotNil(t, restore)
		restore()

		assert.NotContains(t, f.calls, "stopWinNAT")
		assert.NotContains(t, f.calls, "startWinNAT")
		assert.Contains(t, f.calls, "startWSL")
	})

	t.Run("runtime never recovers: returns error and restores before returning", func(t *testing.T) {
		f := &fakeDockerDesktop{engineUp: true}
		ops := f.ops(portBusy, true)
		ops.waitForRuntime = func() error { return errors.New("timed out") }

		restore, err := prepareSchedulerHostPorts(ops)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "container runtime did not become available")
		assert.Equal(t, []string{"checkPorts", "shutdownWSL", "stopWinNAT", "startWinNAT", "startWSL"}, f.calls)

		// Already restored, so the caller must not get a restore func.
		assert.Nil(t, restore)
	})
}

// TestFreeSchedulerHostPorts_NoopOutsideWindowsHostPortMode verifies the
// guard conditions: nothing is touched in slim mode, with a Docker network,
// or off Windows.
func TestFreeSchedulerHostPorts_NoopOutsideWindowsHostPortMode(t *testing.T) {
	for name, info := range map[string]initInfo{
		"slim":           {slimMode: true, runtimeVersion: "1.18.4"},
		"docker network": {dockerNetwork: "dapr-net", runtimeVersion: "1.18.4"},
		"host ports":     {runtimeVersion: "1.18.4"},
	} {
		t.Run(name, func(t *testing.T) {
			if name == "host ports" && runtime.GOOS == daprWindowsOS {
				t.Skip("host-port mode is active on Windows")
			}
			restore, err := freeSchedulerHostPorts(info)
			require.NoError(t, err)
			assert.Nil(t, restore)
		})
	}
}

func TestWaitForContainerRuntime(t *testing.T) {
	t.Run("returns nil once the runtime responds", func(t *testing.T) {
		if runtime.GOOS == daprWindowsOS {
			t.Skip("relies on the POSIX 'true' command")
		}
		truePath, err := exec.LookPath("true")
		if err != nil {
			t.Skip("'true' command not available")
		}
		assert.NoError(t, waitForContainerRuntime(truePath, time.Second, 10*time.Millisecond))
	})

	t.Run("returns error after timeout when the runtime never responds", func(t *testing.T) {
		start := time.Now()
		err := waitForContainerRuntime("dapr-nonexistent-container-runtime", 50*time.Millisecond, 10*time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "did not respond within")
		assert.Less(t, time.Since(start), 2*time.Second)
	})
}
