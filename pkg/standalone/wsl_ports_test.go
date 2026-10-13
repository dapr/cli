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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapr/cli/pkg/ports"
)

// portCause is why the fake host's etcd port (2379) cannot be bound.
type portCause int

const (
	causeNone portCause = iota
	// causeWSLRelay: a listener inside a WSL distro, forwarded to Windows
	// localhost by wslrelay.exe. Freed by `wsl --shutdown`.
	causeWSLRelay
	// causeUnknownOwner: in use, owner could not be determined (e.g. netstat
	// failed) while a WSL distro is running. Freed by `wsl --shutdown` in
	// this model.
	causeUnknownOwner
	// causeUnknownOwnerNoWSL: like causeUnknownOwner, but no WSL distro is
	// running, so shutting WSL down cannot be what frees it.
	causeUnknownOwnerNoWSL
	// causeWinNATReservation: inside a dynamic excluded port range reserved
	// by WinNAT; nobody listens. Freed by stopping WinNAT.
	causeWinNATReservation
	// causePersistentReservation: inside an administered excluded port range
	// (`netsh int ipv4 add excludedportrange`). Nothing the CLI does frees it.
	causePersistentReservation
	// causeRuntimeLinger: Docker Desktop's port proxy still holds the port of
	// a container that was just removed, and releases it after a few scans.
	causeRuntimeLinger
	// causeRuntimeStuck: like causeRuntimeLinger, but never released.
	causeRuntimeStuck
	// causeContainer: another container publishes the port.
	causeContainer
	// causeOtherProcess: a Windows process (e.g. a native etcd) listens.
	causeOtherProcess
	// causeBindError: the bind fails for a reason that is neither "in use"
	// nor "reserved" (e.g. no buffer space). Nothing points at WSL or WinNAT.
	causeBindError
	// causeWSLReacquires: held by wslrelay, and grabbed again by the time the
	// container runtime is back after `wsl --shutdown`.
	causeWSLReacquires
)

// fakeHost models a Windows host with Docker Desktop on the WSL2 backend: its
// engine runs inside the docker-desktop WSL distro, so `wsl --shutdown` kills
// it until Docker Desktop restarts the distro.
type fakeHost struct {
	cause    portCause
	elevated bool
	engineUp bool
	wslUp    bool
	winNATUp bool
	// lingerScans is how many scans a lingering port stays held for.
	lingerScans int
	scans       int
	// clock is virtual time: sleep and scanCost advance it.
	clock    time.Time
	scanCost time.Duration
	calls    []string
}

func newFakeHost(cause portCause) *fakeHost {
	return &fakeHost{cause: cause, elevated: true, engineUp: true, wslUp: true, winNATUp: true, lingerScans: 3}
}

var errEnginePing = errors.New("request returned 500 Internal Server Error for API route and version " +
	"http://%2F%2F.%2Fpipe%2FdockerDesktopLinuxEngine/_ping, check if the server supports the requested API version")

func (f *fakeHost) scan() []ports.Conflict {
	f.calls = append(f.calls, "scan")
	f.scans++
	f.clock = f.clock.Add(f.scanCost)
	inUse := func(holder ports.Holder, process string) []ports.Conflict {
		return []ports.Conflict{{Port: schedulerEtcdPort, Status: ports.InUse, Holder: holder, PID: 4242, Process: process}}
	}
	reserved := []ports.Conflict{{Port: schedulerEtcdPort, Status: ports.Reserved}}

	switch f.cause {
	case causeWSLRelay:
		if f.wslUp {
			return inUse(ports.HolderWSL, "wslrelay.exe")
		}
	case causeUnknownOwner, causeUnknownOwnerNoWSL:
		if f.wslUp {
			return []ports.Conflict{{Port: schedulerEtcdPort, Status: ports.InUse, PID: 4242}}
		}
	case causeWSLReacquires:
		return inUse(ports.HolderWSL, "wslrelay.exe")
	case causeWinNATReservation:
		if f.winNATUp {
			return reserved
		}
	case causePersistentReservation:
		return reserved
	case causeRuntimeLinger:
		if f.scans <= f.lingerScans {
			return inUse(ports.HolderContainerRuntime, "com.docker.backend.exe")
		}
	case causeRuntimeStuck, causeContainer:
		return inUse(ports.HolderContainerRuntime, "com.docker.backend.exe")
	case causeOtherProcess:
		return inUse(ports.HolderOther, "etcd.exe")
	case causeBindError:
		return []ports.Conflict{{Port: schedulerEtcdPort, Status: ports.Unavailable, Err: errors.New("no buffer space available")}}
	}
	return nil
}

func (f *fakeHost) ops() wslPortOps {
	record := func(call string) { f.calls = append(f.calls, call) }
	return wslPortOps{
		elevated:  func() bool { return f.elevated },
		scanPorts: f.scan,
		containerPublishing: func(int) string {
			if f.cause == causeContainer {
				return "my-etcd"
			}
			return ""
		},
		isWSLRunning:    func() bool { return f.wslUp && f.cause != causeUnknownOwnerNoWSL },
		shutdownWSL:     func() error { record("shutdownWSL"); f.wslUp, f.engineUp = false, false; return nil },
		isWinNATRunning: func() bool { return f.winNATUp },
		stopWinNAT:      func() error { record("stopWinNAT"); f.winNATUp = false; return nil },
		startWinNAT:     func() error { record("startWinNAT"); f.winNATUp = true; return nil },
		startWSL:        func() { record("startWSL"); f.wslUp = true },
		waitForRuntime: func() error {
			// Docker Desktop restarts its distro after wsl --shutdown.
			record("waitForRuntime")
			f.engineUp = true
			return nil
		},
		sleep:           func(d time.Duration) { record("sleep"); f.clock = f.clock.Add(d) },
		now:             func() time.Time { return f.clock },
		releaseTimeout:  5 * time.Second,
		releaseInterval: time.Second,
	}
}

// initContainers models Init's parallel container steps (placement,
// scheduler, redis, zipkin): all of them need the engine to be up, and the
// scheduler needs its ports.
func (f *fakeHost) initContainers() error {
	for _, step := range []string{"placement", "scheduler", "redis", "zipkin"} {
		f.calls = append(f.calls, "dockerRun")
		if !f.engineUp {
			return errEnginePing
		}
		if step == "scheduler" && len(f.scanQuiet()) > 0 {
			return errors.New("ports are not available: exposing port TCP 0.0.0.0:2379")
		}
	}
	return nil
}

// scanQuiet scans without recording the call or advancing linger state.
func (f *fakeHost) scanQuiet() []ports.Conflict {
	calls, scans := f.calls, f.scans
	defer func() { f.calls, f.scans = calls, scans }()
	return f.scan()
}

func (f *fakeHost) called(call string) bool {
	for _, c := range f.calls {
		if c == call {
			return true
		}
	}
	return false
}

// TestPrepareSchedulerHostPorts_Causes runs init against every cause of a
// scheduler port being reported as unavailable, and checks the CLI picks the
// remedy that actually frees it, never shuts WSL (and so Docker Desktop's
// engine) down when that cannot help, and leaves WSL/WinNAT as it found them.
func TestPrepareSchedulerHostPorts_Causes(t *testing.T) {
	tests := []struct {
		name        string
		cause       portCause
		nonElevated bool
		winNATDown  bool
		// wantErr is a substring of the expected error, "" for success.
		wantErr string
		// wantWSLShutdown/wantWinNATStop: whether the remedy is expected.
		wantWSLShutdown bool
		wantWinNATStop  bool
	}{
		{name: "ports free", cause: causeNone},
		{name: "held by WSL relay", cause: causeWSLRelay, wantWSLShutdown: true, wantWinNATStop: true},
		{name: "owner unknown, WSL running", cause: causeUnknownOwner, wantWSLShutdown: true, wantWinNATStop: true},
		{
			name: "owner unknown, no WSL distro running: error naming the pid", cause: causeUnknownOwnerNoWSL,
			wantErr: "in use by pid 4242",
		},
		{
			name: "reserved by WinNAT: WinNAT stopped, WSL left alone", cause: causeWinNATReservation,
			wantWinNATStop: true,
		},
		{
			name: "persistently reserved: error after restoring WinNAT", cause: causePersistentReservation,
			wantWinNATStop: true, wantErr: "persistent excluded port range",
		},
		{name: "runtime releasing a removed container's port: waits", cause: causeRuntimeLinger},
		{
			name: "runtime never releases port: error, WSL left alone", cause: causeRuntimeStuck,
			wantErr: "was not released within",
		},
		{
			name: "published by another container: error naming it", cause: causeContainer,
			wantErr: `published by container "my-etcd"`,
		},
		{
			name: "held by another Windows process: error naming it", cause: causeOtherProcess,
			wantErr: "in use by etcd.exe (pid 4242)",
		},
		{
			name: "unexpected bind error: error with the cause, nothing touched", cause: causeBindError,
			wantErr: "no buffer space available",
		},
		{
			name: "WSL re-acquires port after restart: error after restoring", cause: causeWSLReacquires,
			wantWSLShutdown: true, wantWinNATStop: true, wantErr: "could not be freed",
		},
		{
			name: "non-elevated, held by WSL: elevation guidance", cause: causeWSLRelay, nonElevated: true,
			wantErr: "stop and restart WSL",
		},
		{
			name: "non-elevated, reserved: WinNAT guidance, WSL not mentioned as remedy", cause: causeWinNATReservation,
			nonElevated: true, wantErr: "WSL is not shut down",
		},
		{
			name: "non-elevated, persistently reserved: netsh guidance, elevation not suggested", cause: causePersistentReservation,
			nonElevated: true, winNATDown: true, wantErr: "persistent excluded port range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeHost(tt.cause)
			f.elevated = !tt.nonElevated
			f.winNATUp = !tt.winNATDown

			restore, err := prepareSchedulerHostPorts(f.ops())

			assert.Equal(t, tt.wantWSLShutdown, f.called("shutdownWSL"), "WSL shutdown; calls: %v", f.calls)
			assert.Equal(t, tt.wantWinNATStop, f.called("stopWinNAT"), "WinNAT stop; calls: %v", f.calls)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				if tt.nonElevated && tt.winNATDown {
					assert.NotContains(t, err.Error(), "elevated", "elevation cannot lift a persistent exclusion")
				}
				assert.Nil(t, restore, "on error everything is restored before returning")
			} else {
				require.NoError(t, err)
				require.NoError(t, f.initContainers(), "containers must start: engine up and ports free; calls: %v", f.calls)
				assert.Equal(t, tt.wantWSLShutdown || tt.wantWinNATStop, restore != nil)
				if restore != nil {
					restore()
				}
			}

			// Whatever happened, the host is left as it was found.
			assert.True(t, f.engineUp, "Docker engine must be up")
			assert.True(t, f.wslUp, "WSL must be restarted if it was shut down")
			assert.Equal(t, !tt.winNATDown, f.winNATUp, "WinNAT must be left as it was found")
		})
	}
}

// TestPrepareSchedulerHostPorts_DockerDesktopRepro reproduces the report where
// an elevated `dapr init` on Windows with Docker Desktop (WSL2 backend)
// failed with "500 Internal Server Error ... dockerDesktopLinuxEngine/_ping":
// the CLI ran `wsl --shutdown` and then ran `docker run` straight into the
// dead engine.
func TestPrepareSchedulerHostPorts_DockerDesktopRepro(t *testing.T) {
	f := newFakeHost(causeWSLRelay)
	restore, err := prepareSchedulerHostPorts(f.ops())
	require.NoError(t, err)
	require.NotNil(t, restore)
	require.NoError(t, f.initContainers(), "no container may start before Docker Desktop restarts its engine")
	restore()

	assert.Equal(t, []string{
		"scan", "shutdownWSL", "stopWinNAT", "waitForRuntime", "scan",
		"dockerRun", "dockerRun", "dockerRun", "dockerRun",
		"startWinNAT", "startWSL",
	}, f.calls)
}

func TestPrepareSchedulerHostPorts(t *testing.T) {
	t.Run("WinNAT not running is neither stopped nor started", func(t *testing.T) {
		f := newFakeHost(causeWSLRelay)
		f.winNATUp = false
		restore, err := prepareSchedulerHostPorts(f.ops())
		require.NoError(t, err)
		require.NotNil(t, restore)
		restore()

		assert.False(t, f.called("stopWinNAT"))
		assert.False(t, f.called("startWinNAT"))
		assert.True(t, f.called("startWSL"))
	})

	t.Run("reserved port with WinNAT not running: error, nothing touched", func(t *testing.T) {
		f := newFakeHost(causePersistentReservation)
		f.winNATUp = false
		restore, err := prepareSchedulerHostPorts(f.ops())
		require.Error(t, err)
		assert.Nil(t, restore)
		assert.Contains(t, err.Error(), "persistent excluded port range")
		assert.Equal(t, []string{"scan"}, f.calls)
	})

	t.Run("runtime never recovers: returns error and restores before returning", func(t *testing.T) {
		f := newFakeHost(causeWSLRelay)
		ops := f.ops()
		ops.waitForRuntime = func() error { return errors.New("timed out") }

		restore, err := prepareSchedulerHostPorts(ops)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "container runtime did not become available")
		assert.Equal(t, []string{"scan", "shutdownWSL", "stopWinNAT", "startWinNAT", "startWSL"}, f.calls)

		// Already restored, so the caller must not get a restore func.
		assert.Nil(t, restore)
	})

	t.Run("lingering runtime port: polls until released, bounded by the timeout", func(t *testing.T) {
		f := newFakeHost(causeRuntimeLinger)
		restore, err := prepareSchedulerHostPorts(f.ops())
		require.NoError(t, err)
		assert.Nil(t, restore)
		// Held for 3 scans: the initial one and two polls; free on the third poll.
		assert.Equal(t, []string{"scan", "sleep", "scan", "sleep", "scan", "sleep", "scan"}, f.calls)

		f = newFakeHost(causeRuntimeStuck)
		_, err = prepareSchedulerHostPorts(f.ops())
		require.Error(t, err)
		sleeps := 0
		for _, c := range f.calls {
			if c == "sleep" {
				sleeps++
			}
		}
		assert.Equal(t, 5, sleeps, "releaseTimeout / releaseInterval polls")
	})

	t.Run("release timeout is wall-clock: slow scans mean fewer polls", func(t *testing.T) {
		f := newFakeHost(causeRuntimeStuck)
		f.scanCost = 2 * time.Second // netstat + tasklist per scan
		_, err := prepareSchedulerHostPorts(f.ops())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "was not released within 5s")
		sleeps := 0
		for _, c := range f.calls {
			if c == "sleep" {
				sleeps++
			}
		}
		// The wait starts after the initial scan; each poll is sleep 1s +
		// scan 2s, so polls end at 3s and 6s and the third is not started.
		// Counting sleeps instead of time would have polled five times (15s).
		assert.Equal(t, 2, sleeps)
		assert.Equal(t, 8*time.Second, f.clock.Sub(time.Time{}), "initial scan 2s + two polls of 3s")
	})

	t.Run("a container publishing any runtime-held port is named without waiting", func(t *testing.T) {
		f := newFakeHost(causeRuntimeLinger)
		ops := f.ops()
		scan := ops.scanPorts
		ops.scanPorts = func() []ports.Conflict {
			// 6060 lingers from a removed container; 2379 belongs to a running one.
			return append(scan(), ports.Conflict{Port: schedulerEtcdPort, Status: ports.InUse,
				Holder: ports.HolderContainerRuntime, Process: "com.docker.backend.exe"})
		}
		ops.containerPublishing = func(port int) string {
			if port == schedulerEtcdPort {
				return "my-etcd"
			}
			return ""
		}
		f.cause = causeRuntimeStuck
		_, err := prepareSchedulerHostPorts(ops)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `published by container "my-etcd"`)
		assert.NotContains(t, f.calls, "sleep")
	})

	t.Run("WSL relay and WinNAT reservation together: both remedies applied", func(t *testing.T) {
		f := newFakeHost(causeWSLRelay)
		ops := f.ops()
		scan := ops.scanPorts
		ops.scanPorts = func() []ports.Conflict {
			conflicts := scan()
			if f.winNATUp {
				conflicts = append(conflicts, ports.Conflict{Port: schedulerHealthPort, Status: ports.Reserved})
			}
			return conflicts
		}
		restore, err := prepareSchedulerHostPorts(ops)
		require.NoError(t, err)
		require.NotNil(t, restore)
		restore()
		assert.True(t, f.called("shutdownWSL"))
		assert.True(t, f.called("stopWinNAT"))
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

	t.Run("kills a hung info call at the timeout", func(t *testing.T) {
		if runtime.GOOS == daprWindowsOS {
			t.Skip("relies on a POSIX shell script")
		}
		// A fake runtime whose `info` never returns, like a Docker Desktop
		// pipe that accepts the connection but stalls while WSL2 restarts.
		hung := filepath.Join(t.TempDir(), "hung-runtime")
		require.NoError(t, os.WriteFile(hung, []byte("#!/bin/sh\nexec sleep 30\n"), 0o755)) //nolint:gosec

		start := time.Now()
		err := waitForContainerRuntime(hung, 200*time.Millisecond, 10*time.Millisecond)
		require.Error(t, err)
		assert.Less(t, time.Since(start), 5*time.Second)
	})

	t.Run("returns error after timeout when the runtime never responds", func(t *testing.T) {
		start := time.Now()
		err := waitForContainerRuntime("dapr-nonexistent-container-runtime", 50*time.Millisecond, 10*time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "did not respond within")
		assert.Less(t, time.Since(start), 2*time.Second)
	})
}

func TestFirstInitError(t *testing.T) {
	stepErr := errors.New("placement failed")

	// run starts a fast failing step and a slow succeeding step, closing the
	// channel once both are done, as Init does. It returns the error and
	// whether the slow step had finished when firstInitError returned.
	run := func(waitForAll bool) (error, bool) {
		errorChan := make(chan error)
		var wg sync.WaitGroup
		var slowDone atomic.Bool
		wg.Add(2)
		go func() { defer wg.Done(); errorChan <- stepErr }()
		go func() {
			defer wg.Done()
			time.Sleep(100 * time.Millisecond)
			slowDone.Store(true)
			errorChan <- nil
		}()
		go func() { wg.Wait(); close(errorChan) }()

		err := firstInitError(errorChan, waitForAll)
		done := slowDone.Load()
		if !waitForAll {
			for range errorChan { //nolint:revive // drain so the test goroutines exit
			}
		}
		return err, done
	}

	t.Run("returns on the first error when nothing needs restoring", func(t *testing.T) {
		err, slowDone := run(false)
		require.ErrorIs(t, err, stepErr)
		assert.False(t, slowDone)
	})

	t.Run("waits for every step when WSL must be restored", func(t *testing.T) {
		err, slowDone := run(true)
		require.ErrorIs(t, err, stepErr)
		assert.True(t, slowDone, "restore must not run while a step is still in progress")
	})

	t.Run("returns nil when every step succeeds", func(t *testing.T) {
		errorChan := make(chan error, 2)
		errorChan <- nil
		errorChan <- nil
		close(errorChan)
		assert.NoError(t, firstInitError(errorChan, true))
	})
}
