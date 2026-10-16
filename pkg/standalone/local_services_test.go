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
	"bytes"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fakeServiceEnv = "DAPR_CLI_TEST_FAKE_LOCAL_SERVICE"

// TestFakeLocalService is not a test. It is the process that the tests below
// start in place of a placement or scheduler binary. It serves /healthz on the
// port after --healthz-port and exits on an interrupt.
func TestFakeLocalService(t *testing.T) {
	mode := os.Getenv(fakeServiceEnv)
	if mode == "" {
		t.Skip("helper process for the local services tests")
	}
	if mode == "exit" {
		os.Exit(3)
	}

	healthzPort := ""
	for i, arg := range os.Args {
		if arg == "--healthz-port" && i+1 < len(os.Args) {
			healthzPort = os.Args[i+1]
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:"+healthzPort)
	if err != nil {
		os.Exit(4)
	}
	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	os.Exit(0)
}

func fakeServiceSpec(t *testing.T) localServiceSpec {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	return localServiceSpec{
		name:        schedulerServiceFilePrefix,
		port:        port,
		healthzPort: port,
		args:        []string{"-test.run=^TestFakeLocalService$", "--", "--healthz-port", strconv.Itoa(port)},
	}
}

func TestStartLocalService(t *testing.T) {
	t.Run("start, wait until healthy, stop", func(t *testing.T) {
		t.Setenv(fakeServiceEnv, "serve")
		spec := fakeServiceSpec(t)
		logPath := filepath.Join(t.TempDir(), "logs", "scheduler.log")

		s, err := startLocalService(os.Args[0], logPath, spec)
		require.NoError(t, err)
		require.NoError(t, s.waitHealthy(30*time.Second))
		assert.FileExists(t, logPath)

		s.stop(10 * time.Second)
		select {
		case <-s.exited:
		default:
			t.Fatal("the service still runs after stop")
		}
	})

	t.Run("a process that exits is not healthy", func(t *testing.T) {
		t.Setenv(fakeServiceEnv, "exit")
		spec := fakeServiceSpec(t)

		s, err := startLocalService(os.Args[0], filepath.Join(t.TempDir(), "scheduler.log"), spec)
		require.NoError(t, err)
		err = s.waitHealthy(30 * time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "the process exited")
	})

	t.Run("the log file keeps the output of earlier runs", func(t *testing.T) {
		t.Setenv(fakeServiceEnv, "exit")
		logPath := filepath.Join(t.TempDir(), "scheduler.log")
		require.NoError(t, os.WriteFile(logPath, []byte("output of another run\n"), 0o600))

		s, err := startLocalService(os.Args[0], logPath, fakeServiceSpec(t))
		require.NoError(t, err)
		<-s.exited

		content, err := os.ReadFile(logPath)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(content), "output of another run\n"), "log file: %q", content)
	})

	t.Run("a binary that does not exist does not start", func(t *testing.T) {
		_, err := startLocalService(filepath.Join(t.TempDir(), "scheduler"), filepath.Join(t.TempDir(), "scheduler.log"), fakeServiceSpec(t))
		require.Error(t, err)
	})
}

func TestPlacementServiceSpec(t *testing.T) {
	t.Run("1.18 listens on localhost only", func(t *testing.T) {
		spec := placementServiceSpec("1.18.7", 50005)
		assert.Equal(t, placementServiceFilePrefix, spec.name)
		assert.Equal(t, 50005, spec.port)
		assert.Equal(t, healthPort, spec.healthzPort)
		assert.Equal(t, []int{8201, 58080, 59090}, spec.otherPorts)
		assert.Equal(t, []string{
			"--port", "50005",
			"--healthz-port", "58080",
			"--metrics-port", "59090",
			"--listen-address", "127.0.0.1",
			"--healthz-listen-address", "127.0.0.1",
			"--metrics-listen-address", "127.0.0.1",
		}, spec.args)
	})

	t.Run("1.13 has no listen address flags", func(t *testing.T) {
		spec := placementServiceSpec("1.13.6", 50005)
		assert.Equal(t, []string{"--port", "50005", "--healthz-port", "58080", "--metrics-port", "59090"}, spec.args)
	})

	t.Run("edge has listen address flags", func(t *testing.T) {
		assert.Contains(t, placementServiceSpec("edge", 50005).args, "--listen-address")
	})
}

func TestSchedulerServiceSpec(t *testing.T) {
	dataDir := filepath.Join("home", ".dapr", "scheduler", "data")

	t.Run("1.18 broadcasts localhost", func(t *testing.T) {
		spec := schedulerServiceSpec("1.18.7", dataDir, 50006, false)
		assert.Equal(t, schedulerServiceFilePrefix, spec.name)
		assert.Equal(t, 50006, spec.port)
		assert.Equal(t, schedulerHealthPort, spec.healthzPort)
		assert.Equal(t, []int{2379, 2380, 58081, 59091}, spec.otherPorts)
		assert.Equal(t, []string{
			"--port", "50006",
			"--listen-address", "127.0.0.1",
			"--healthz-port", "58081",
			"--healthz-listen-address", "127.0.0.1",
			"--metrics-port", "59091",
			"--metrics-listen-address", "127.0.0.1",
			"--etcd-data-dir", dataDir,
			"--override-broadcast-host-port=localhost:50006",
		}, spec.args)
	})

	t.Run("1.14 has no broadcast override", func(t *testing.T) {
		spec := schedulerServiceSpec("1.14.4", dataDir, 50006, false)
		assert.NotContains(t, spec.args, "--override-broadcast-host-port=localhost:50006")
	})

	t.Run("1.19 serves placement when asked", func(t *testing.T) {
		spec := schedulerServiceSpec("1.19.0-rc.2", dataDir, 50006, true)
		assert.Contains(t, spec.args, "--placement-enabled=true")
		spec = schedulerServiceSpec("1.19.0-rc.2", dataDir, 50006, false)
		assert.NotContains(t, spec.args, "--placement-enabled=true")
	})
}

// newTestLocalServices returns LocalServices for a fake installation with the given binaries.
func newTestLocalServices(t *testing.T, version string, listening bool, binaries ...string) (*LocalServices, *bytes.Buffer) {
	t.Helper()
	installDir := t.TempDir()
	binDir := getDaprBinPath(installDir)
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	for _, b := range binaries {
		// Not executable, so a start fails.
		require.NoError(t, os.WriteFile(binaryFilePathWithDir(binDir, b), []byte("not a binary"), 0o600))
	}

	out := &bytes.Buffer{}
	return &LocalServices{
		binDir:           binDir,
		logsDir:          getDaprLogsPath(installDir),
		schedulerDataDir: getDaprSchedulerDataPath(installDir),
		runtimeVersion:   version,
		handled:          map[string]bool{},
		isListening:      func(string) bool { return listening },
		canListen:        func(string) bool { return true },
		out:              out,
		placementPort:    50005,
		schedulerPort:    50006,
	}, out
}

func specNames(specs []localServiceSpec) []string {
	names := []string{}
	for _, s := range specs {
		names = append(names, s.name)
	}
	return names
}

func TestLocalServicesPlan(t *testing.T) {
	t.Run("docker install has no binaries, so nothing starts", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "1.18.7", false)
		assert.Empty(t, l.plan(true, true))
	})

	t.Run("slim install starts placement and scheduler", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "1.18.7", false, "placement", "scheduler")
		specs := l.plan(true, true)
		assert.Equal(t, []string{"placement", "scheduler"}, specNames(specs))
		assert.NotContains(t, specs[1].args, "--placement-enabled=true")
	})

	t.Run("slim install with scheduler placement starts only the scheduler", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "1.19.0", false, "scheduler")
		specs := l.plan(true, true)
		assert.Equal(t, []string{"scheduler"}, specNames(specs))
		assert.Contains(t, specs[0].args, "--placement-enabled=true")
	})

	t.Run("no placement binary on 1.18 does not enable scheduler placement", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "1.18.7", false, "scheduler")
		specs := l.plan(true, true)
		assert.Equal(t, []string{"scheduler"}, specNames(specs))
		assert.NotContains(t, specs[0].args, "--placement-enabled=true")
	})

	t.Run("a user address keeps the service out", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "1.18.7", false, "placement", "scheduler")
		assert.Equal(t, []string{"placement"}, specNames(l.plan(true, false)))

		l, _ = newTestLocalServices(t, "1.18.7", false, "placement", "scheduler")
		assert.Equal(t, []string{"scheduler"}, specNames(l.plan(false, true)))
	})

	t.Run("each service is looked at once", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "1.18.7", false, "placement", "scheduler")
		assert.Len(t, l.plan(true, true), 2)
		assert.Empty(t, l.plan(true, true))
	})

	t.Run("no daprd version, nothing starts", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "", false, "placement", "scheduler")
		assert.Empty(t, l.plan(true, true))
	})

	t.Run("no daprd binary, nothing starts", func(t *testing.T) {
		// GetRuntimeVersion returns "n/a" and an error when daprd is not installed.
		runtimePath := t.TempDir()
		binDir := getDaprBinPath(filepath.Join(runtimePath, DefaultDaprDirName))
		require.NoError(t, os.MkdirAll(binDir, 0o755))
		for _, b := range []string{placementServiceFilePrefix, schedulerServiceFilePrefix} {
			require.NoError(t, os.WriteFile(binaryFilePathWithDir(binDir, b), []byte("not a binary"), 0o600))
		}

		l, err := NewLocalServices(runtimePath)
		require.NoError(t, err)
		assert.Empty(t, l.runtimeVersion)
		assert.Empty(t, l.plan(true, true))
	})
}

func TestLocalServicesStart(t *testing.T) {
	t.Run("a running service is used, not started", func(t *testing.T) {
		l, out := newTestLocalServices(t, "1.18.7", true, "placement", "scheduler")
		l.Start(true, true)
		assert.Empty(t, l.started)
		assert.Contains(t, out.String(), "Using the placement service that runs on localhost:50005")
		assert.Contains(t, out.String(), "Using the scheduler service that runs on localhost:50006")
	})

	t.Run("a service that cannot start gives a warning", func(t *testing.T) {
		l, out := newTestLocalServices(t, "1.18.7", false, "scheduler")
		l.Start(false, true)
		assert.Empty(t, l.started)
		assert.Contains(t, out.String(), "Could not start the scheduler service")
	})

	t.Run("a service that another run started at the same moment is used", func(t *testing.T) {
		l, out := newTestLocalServices(t, "1.18.7", false, "scheduler")
		// Free when Start looks, in use when the start fails.
		calls := 0
		l.isListening = func(string) bool {
			calls++
			return calls > 1
		}
		l.Start(false, true)
		assert.Empty(t, l.started)
		assert.Contains(t, out.String(), "Using the scheduler service that runs on localhost:50006")
		assert.NotContains(t, out.String(), "Could not start")
	})

	t.Run("a service with another port in use does not start", func(t *testing.T) {
		for _, tc := range []struct {
			service string
			port    string
		}{
			{"scheduler", "2379"},
			{"scheduler", "2380"},
			{"placement", "8201"},
		} {
			l, out := newTestLocalServices(t, "1.18.7", false, tc.service)
			var checked []string
			l.canListen = func(address string) bool {
				checked = append(checked, address)
				return address != "127.0.0.1:"+tc.port
			}
			l.Start(tc.service == "placement", tc.service == "scheduler")
			assert.Empty(t, l.started)
			assert.Contains(t, checked, "127.0.0.1:"+tc.port)
			assert.Contains(t, out.String(), "Did not start the "+tc.service+" service, because another process uses port "+tc.port+".")
			// The binary of the test is not executable, so a start attempt gives "Could not start".
			assert.NotContains(t, out.String(), "Could not start")
		}
	})

	t.Run("the other ports of a running service are not checked", func(t *testing.T) {
		l, out := newTestLocalServices(t, "1.18.7", true, "placement", "scheduler")
		l.canListen = func(address string) bool {
			t.Errorf("canListen(%q) called for a running service", address)
			return false
		}
		l.Start(true, true)
		assert.Contains(t, out.String(), "Using the placement service that runs on localhost:50005")
		assert.Contains(t, out.String(), "Using the scheduler service that runs on localhost:50006")
	})

	t.Run("a port in use by a service that another run started at the same moment", func(t *testing.T) {
		l, out := newTestLocalServices(t, "1.18.7", false, "scheduler")
		// Free when Start looks, in use when the port check fails.
		calls := 0
		l.isListening = func(string) bool {
			calls++
			return calls > 1
		}
		l.canListen = func(string) bool { return false }
		l.Start(false, true)
		assert.Empty(t, l.started)
		assert.Contains(t, out.String(), "Using the scheduler service that runs on localhost:50006")
		assert.NotContains(t, out.String(), "Did not start")
	})

	t.Run("nil does nothing", func(t *testing.T) {
		var l *LocalServices
		l.Start(true, true)
		l.Stop()
		assert.Empty(t, l.SchedulerWarning(nil))
	})
}

func TestLocalServicesSchedulerWarning(t *testing.T) {
	addr := "localhost:50006"
	empty := ""

	t.Run("no warning when the scheduler answers", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "1.18.7", true, "scheduler")
		assert.Empty(t, l.SchedulerWarning(&addr))
	})

	t.Run("no warning when the scheduler is disabled", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "1.18.7", false, "scheduler")
		assert.Empty(t, l.SchedulerWarning(nil))
		assert.Empty(t, l.SchedulerWarning(&empty))
	})

	t.Run("no warning for a runtime without a scheduler", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "1.13.6", false)
		assert.Empty(t, l.SchedulerWarning(&addr))
	})

	t.Run("slim install", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "1.18.7", false, "scheduler")
		warning := l.SchedulerWarning(&addr)
		assert.Contains(t, warning, "No scheduler service answers on localhost:50006.")
		assert.Contains(t, warning, "Workflows, actor reminders and the Jobs API")
		assert.NotContains(t, warning, "container")
	})

	t.Run("docker install", func(t *testing.T) {
		l, _ := newTestLocalServices(t, "1.18.7", false)
		warning := l.SchedulerWarning(&addr)
		assert.Contains(t, warning, "No scheduler service answers on localhost:50006.")
		assert.Contains(t, warning, "dapr_scheduler container")
	})
}

func TestCanListen(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	assert.False(t, canListen(addr))
	require.NoError(t, ln.Close())
	assert.True(t, canListen(addr))
}

func TestIsListening(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	assert.True(t, isListening(addr))
	require.NoError(t, ln.Close())
	assert.False(t, isListening(addr))
}
