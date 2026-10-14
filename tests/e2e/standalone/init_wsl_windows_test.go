//go:build windows && e2e && !template

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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dapr/cli/pkg/ports"
	"github.com/dapr/cli/tests/e2e/common"
	"github.com/dapr/cli/tests/e2e/spawn"
	"github.com/dapr/cli/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// wslShutdownMessage is printed by `dapr init` when it shuts WSL down to
	// free a scheduler host port.
	wslShutdownMessage = "Temporarily shutting down WSL"

	// schedulerEtcdHostPort is the host port of the scheduler's etcd client.
	schedulerEtcdHostPort = 2379

	// wslShutdownTestEnvVar opts in to TestStandaloneInitWSLHoldsSchedulerPort,
	// which makes `dapr init` shut down every WSL distro on the machine.
	wslShutdownTestEnvVar = "DAPR_E2E_WSL_SHUTDOWN"
)

// TestStandaloneInitAfterUninstallKeepsWSL runs init straight after uninstall,
// when the container runtime may not have released the ports of the removed
// containers yet. Nothing in WSL holds a scheduler port, so init must succeed
// without shutting WSL down, which would stop Docker Desktop's engine.
func TestStandaloneInitAfterUninstallKeepsWSL(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping: slim mode starts no containers")
	}
	daprRuntimeVersion := common.GetVersionsFromEnv(t, false)

	for range 3 {
		must(t, cmdUninstall, "failed to uninstall Dapr")

		output, err := cmdInit("--runtime-version", daprRuntimeVersion)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.NotContains(t, output, wslShutdownMessage)

		verifyContainers(t, daprRuntimeVersion)
	}
}

// TestStandaloneInitWSLHoldsSchedulerPort holds the scheduler's etcd host port
// from a WSL distro. WSL's localhost relay forwards the listener to the
// Windows host, where it blocks the scheduler container. Init must attribute
// the port to WSL, shut WSL down to free it, wait for the container runtime
// to come back and complete.
//
// It needs an elevated terminal and a container runtime on the WSL2 backend
// (Docker Desktop).
func TestStandaloneInitWSLHoldsSchedulerPort(t *testing.T) {
	if isSlimMode() {
		t.Skip("skipping: slim mode starts no containers")
	}
	if os.Getenv(wslShutdownTestEnvVar) != "true" {
		t.Skipf("skipping: this test shuts WSL down; set %s=true to run it", wslShutdownTestEnvVar)
	}
	daprRuntimeVersion := common.GetVersionsFromEnv(t, false)
	// wsl.exe prints UTF-16 unless told otherwise.
	t.Setenv("WSL_UTF8", "1")

	t.Cleanup(func() {
		// Reinstall Dapr so subsequent tests still have a working installation.
		cmdUninstall()
		ensureDaprInstallation(t)
	})

	// A distro of our own, so that the listener is not the container runtime's.
	const distro = "dapr-e2e-port-holder"
	const image = "alpine:3"
	dir := t.TempDir()
	rootfs := filepath.Join(dir, "rootfs.tar")
	runtimeCmd := utils.GetContainerRuntimeCmd(containerRuntime())
	mustRun(t, runtimeCmd, "pull", image)
	containerID := mustRun(t, runtimeCmd, "create", image)
	mustRun(t, runtimeCmd, "export", "--output", rootfs, containerID)
	mustRun(t, runtimeCmd, "rm", containerID)
	mustRun(t, "wsl", "--import", distro, filepath.Join(dir, distro), rootfs, "--version", "2")
	t.Cleanup(func() {
		if output, err := spawn.Command("wsl", "--unregister", distro); err != nil {
			t.Logf("failed to unregister WSL distro %s: %v: %s", distro, err, output)
		}
	})

	// The relay binds the loopback address of the listener's address family:
	// 127.0.0.1 for an IPv4 listener, [::1] for an IPv6 one.
	for name, bindAddress := range map[string]string{"IPv4 listener": "0.0.0.0", "IPv6 listener": "::"} {
		t.Run(name, func(t *testing.T) {
			must(t, cmdUninstall, "failed to uninstall Dapr")
			require.Eventually(t, func() bool {
				return len(ports.Scan(nil, schedulerEtcdHostPort)) == 0
			}, time.Minute, time.Second, "port %d was not released after uninstall", schedulerEtcdHostPort)

			holder := exec.Command("wsl", "-d", distro, "--exec",
				"/usr/bin/nc", "-lk", "-s", bindAddress, "-p", strconv.Itoa(schedulerEtcdHostPort), "-e", "/bin/cat")
			require.NoError(t, holder.Start(), "failed to start a listener in WSL")
			t.Cleanup(func() {
				holder.Process.Kill()
				holder.Wait()
			})

			var conflicts []ports.Conflict
			if !assert.Eventually(t, func() bool {
				conflicts = ports.Scan(ports.NewOwnerLookup(), schedulerEtcdHostPort)
				return len(conflicts) > 0
			}, time.Minute, time.Second, "the WSL listener on port %d never became visible on the Windows host", schedulerEtcdHostPort) {
				// Tell a listener that never started from one that was not forwarded.
				inDistro, _ := spawn.Command("wsl", "-d", distro, "--exec", "/bin/netstat", "-ltn")
				t.Logf("listeners inside the %s distro:\n%s", distro, inDistro)
				onHost, _ := spawn.Command("netstat", "-ano")
				for _, line := range strings.Split(onHost, "\n") {
					if strings.Contains(line, ":"+strconv.Itoa(schedulerEtcdHostPort)+" ") {
						t.Logf("listener on the Windows host: %s", strings.TrimSpace(line))
					}
				}
				t.FailNow()
			}
			t.Logf("before init: %s", conflicts[0])
			require.Equal(t, ports.HolderWSL, conflicts[0].Holder, "a port forwarded from a WSL distro must be attributed to WSL")

			winNATWasRunning := ports.IsWinNATRunning()

			output, err := cmdInit("--runtime-version", daprRuntimeVersion)
			t.Log(output)
			require.NoError(t, err, "init failed")
			assert.Contains(t, output, wslShutdownMessage)

			verifyContainers(t, daprRuntimeVersion)

			// What init stopped must be usable again once it returns.
			if winNATWasRunning {
				assert.True(t, ports.IsWinNATRunning(), "the Windows NAT service was running before init and is stopped after it")
			}
			mustRun(t, "wsl", "-d", distro, "--exec", "/bin/true")
		})
	}
}

// mustRun runs a command, fails the test if it fails, and returns its output.
func mustRun(t *testing.T, name string, args ...string) string {
	t.Helper()
	output, err := spawn.Command(name, args...)
	require.NoError(t, err, "%s %s: %s", name, strings.Join(args, " "), output)
	return strings.TrimSpace(output)
}
