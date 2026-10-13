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
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/dapr/cli/pkg/ports"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

const (
	// schedulerWindowsGRPCHostPort is the host port the scheduler's gRPC port
	// is published on for Windows (50006 falls in the Hyper-V excluded range).
	schedulerWindowsGRPCHostPort = 6060

	// containerRuntimeRecoveryTimeout bounds how long we wait for the container
	// runtime to come back after `wsl --shutdown`. Docker Desktop on the WSL2
	// backend restarts its docker-desktop distro automatically, which usually
	// takes 10-30s.
	containerRuntimeRecoveryTimeout  = 90 * time.Second
	containerRuntimeRecoveryInterval = 2 * time.Second

	// portReleaseTimeout bounds how long we wait for the container runtime to
	// release a port of a container that was just removed (e.g. `dapr
	// uninstall` immediately followed by `dapr init`).
	portReleaseTimeout  = 15 * time.Second
	portReleaseInterval = 500 * time.Millisecond
)

// wslPortOps holds the operations used to free the scheduler's host ports on
// Windows. Fields are injectable for testing.
type wslPortOps struct {
	elevated  func() bool
	scanPorts func() []ports.Conflict
	// containerPublishing returns the container publishing a host port, or "".
	containerPublishing func(port int) string
	isWSLRunning        func() bool
	shutdownWSL         func() error
	isWinNATRunning     func() bool
	stopWinNAT          func() error
	startWinNAT         func() error
	startWSL            func()
	waitForRuntime      func() error
	sleep               func(time.Duration)
	now                 func() time.Time
	releaseTimeout      time.Duration
	releaseInterval     time.Duration
}

func defaultWSLPortOps(runtimeCmd string, grpcPort int) wslPortOps {
	return wslPortOps{
		elevated: isWindowsElevated,
		scanPorts: func() []ports.Conflict {
			// One owner lookup per scan: it runs netstat once for all ports.
			return ports.Scan(ports.NewOwnerLookup(), schedulerHostPorts(grpcPort)...)
		},
		containerPublishing: func(port int) string { return containerPublishingPort(runtimeCmd, port) },
		isWSLRunning:        isWSLRunning,
		shutdownWSL:         shutdownWSL,
		isWinNATRunning:     ports.IsWinNATRunning,
		stopWinNAT:          stopWinNAT,
		startWinNAT:         startWinNAT,
		startWSL:            startWSLBackground,
		waitForRuntime: func() error {
			return waitForContainerRuntime(runtimeCmd, containerRuntimeRecoveryTimeout, containerRuntimeRecoveryInterval)
		},
		sleep:           time.Sleep,
		now:             time.Now,
		releaseTimeout:  portReleaseTimeout,
		releaseInterval: portReleaseInterval,
	}
}

// schedulerHostPorts returns the host ports the scheduler publishes. grpcPort
// is the platform-specific gRPC port (50006 on Linux/Mac, 6060 on Windows).
func schedulerHostPorts(grpcPort int) []int {
	return []int{grpcPort, schedulerEtcdPort, schedulerHealthPort, schedulerMetricPort}
}

// freeSchedulerHostPorts makes sure the scheduler's host ports are free
// before Init starts any container. It must run before the init steps are
// started, because shutting down WSL also stops Docker Desktop's WSL2 engine,
// which every container step depends on.
//
// The returned restore func restarts WinNAT/WSL and must be called once all
// init steps have finished. It is nil when nothing was stopped.
func freeSchedulerHostPorts(info initInfo) (restore func(), err error) {
	if info.slimMode || info.dockerNetwork != "" || runtime.GOOS != daprWindowsOS || !isWSLAvailable() {
		return nil, nil
	}

	// Leave these error cases to runSchedulerService, which reports them.
	if hasScheduler, schedErr := isSchedulerIncluded(info.runtimeVersion); schedErr != nil || !hasScheduler {
		return nil, nil
	}
	runtimeCmd := utils.GetContainerRuntimeCmd(info.containerRuntime)
	schedulerContainerName := utils.CreateContainerName(DaprSchedulerContainerName, info.dockerNetwork)
	if exists, existsErr := confirmContainerIsRunningOrExists(schedulerContainerName, false, runtimeCmd); existsErr != nil || exists {
		return nil, nil
	}

	return prepareSchedulerHostPorts(defaultWSLPortOps(runtimeCmd, schedulerWindowsGRPCHostPort))
}

// prepareSchedulerHostPorts frees the scheduler's host ports if, and only if,
// one of them cannot be bound, choosing the remedy from why it cannot be:
//
//   - held by the container runtime: a container publishes it (error naming
//     the container), or a just-removed container's port has not been
//     released yet (wait for it). WSL is never shut down for this, as that
//     would stop Docker Desktop's engine without freeing anything.
//   - held by another process: error naming the process; nothing is touched.
//   - any other bind error: error with the cause; nothing is touched.
//   - held by a process that could not be identified: treated as held by
//     WSL only while a WSL distribution is running; otherwise an error
//     naming the pid, as shutting WSL down could not free it.
//   - reserved by Windows (excluded port range, nobody listening): stop
//     WinNAT, which releases its dynamic reservations. WSL is not shut down.
//   - held by WSL: shut down WSL and stop WinNAT, then wait for the
//     container runtime to come back.
//
// Stopping WSL or WinNAT requires elevation; without it a guidance error is
// returned. After stopping them the ports are scanned again, and WinNAT/WSL
// are restored before returning an error if any port is still unavailable.
// On success the caller owns the returned restore func, which is nil when
// nothing was stopped.
func prepareSchedulerHostPorts(ops wslPortOps) (restore func(), err error) {
	conflicts := ops.scanPorts()
	if len(conflicts) == 0 {
		return nil, nil
	}

	if hasConflict(conflicts, isRuntimeHeld) {
		for _, c := range conflicts {
			if !isRuntimeHeld(c) {
				continue
			}
			if name := ops.containerPublishing(c.Port); name != "" {
				return nil, fmt.Errorf("failed to start scheduler service: %s, which is published by container %q.\n\n"+
					"Stop or remove that container and re-run 'dapr init'", c, name)
			}
		}
		c, _ := firstConflict(conflicts, isRuntimeHeld)
		print.InfoStatusEvent(os.Stdout, "Waiting for the container runtime to release port %d...", c.Port)
		conflicts = waitForRuntimeRelease(ops, conflicts)
		if c, ok := firstConflict(conflicts, isRuntimeHeld); ok {
			return nil, fmt.Errorf("failed to start scheduler service: %s and was not released within %s.\n\n"+
				"No container publishes it, so the container runtime is still holding it. "+
				"Restart Docker Desktop (or your container runtime) and re-run 'dapr init'", c, ops.releaseTimeout)
		}
		if len(conflicts) == 0 {
			return nil, nil
		}
	}

	if c, ok := firstConflict(conflicts, isOtherProcessHeld); ok {
		return nil, fmt.Errorf("failed to start scheduler service: %s.\n\n"+
			"Stop that process, or free the port, and re-run 'dapr init'", c)
	}

	// A bind error that is neither "in use" nor "reserved" says nothing about
	// WSL or WinNAT, so stopping them would be a guess.
	if c, ok := firstConflict(conflicts, isUnexpectedBindError); ok {
		return nil, fmt.Errorf("failed to start scheduler service: %s", c)
	}

	// Shutting WSL down for an owner we could not identify is only justified
	// while a WSL distribution is running.
	if c, ok := firstConflict(conflicts, isUnknownHeld); ok && !ops.isWSLRunning() {
		return nil, fmt.Errorf("failed to start scheduler service: %s.\n\n"+
			"The owning process could not be identified and no WSL distribution is running, "+
			"so shutting down WSL would not free the port. Find the process with\n"+
			"  netstat -ano | findstr :%d\n"+
			"stop it, and re-run 'dapr init'", c, c.Port)
	}

	needWSLShutdown := hasConflict(conflicts, isWSLOrUnknownHeld)
	if !ops.elevated() {
		return nil, nonElevatedPortError(ops, conflicts, needWSLShutdown)
	}

	wslShutDown := false
	if needWSLShutdown {
		print.InfoStatusEvent(os.Stdout, "Scheduler %s. Temporarily shutting down WSL to free it...", firstWSLConflict(conflicts))
		if wslErr := ops.shutdownWSL(); wslErr != nil {
			print.WarningStatusEvent(os.Stdout, "Failed to shut down WSL: %v. Continuing...", wslErr)
		} else {
			wslShutDown = true
		}
	}

	winNATStopped := false
	if ops.isWinNATRunning() {
		print.InfoStatusEvent(os.Stdout, "Temporarily stopping Windows NAT service to free scheduler ports...")
		if stopErr := ops.stopWinNAT(); stopErr != nil {
			print.WarningStatusEvent(os.Stdout, "Failed to stop Windows NAT service: %v. Continuing...", stopErr)
		} else {
			winNATStopped = true
		}
	}

	if !wslShutDown && !winNATStopped {
		return nil, stillUnavailableError(conflicts)
	}

	restore = func() {
		if winNATStopped {
			if startErr := ops.startWinNAT(); startErr != nil {
				print.WarningStatusEvent(os.Stdout, "Failed to restart Windows NAT service: %v", startErr)
			}
		}
		if wslShutDown {
			print.InfoStatusEvent(os.Stdout, "Restarting WSL...")
			ops.startWSL()
		}
	}

	if wslShutDown {
		print.InfoStatusEvent(os.Stdout, "Waiting for the container runtime to become available after WSL shutdown...")
		if waitErr := ops.waitForRuntime(); waitErr != nil {
			restore()
			return nil, fmt.Errorf(
				"failed to start scheduler service: container runtime did not become available after shutting down WSL: %w\n\n"+
					"If you are using Docker Desktop, wait until it reports that the engine is running and re-run 'dapr init'",
				waitErr)
		}
	}

	// Re-check: a port reserved persistently, or re-acquired while the
	// container runtime restarted, would otherwise surface as an opaque
	// `docker run` failure.
	if remaining := ops.scanPorts(); len(remaining) > 0 {
		restore()
		return nil, stillUnavailableError(remaining)
	}

	return restore, nil
}

// waitForRuntimeRelease rescans until no port is held by the container
// runtime or the release timeout elapses (wall-clock: each scan runs netstat
// and tasklist, which take a while on Windows), returning the last scan.
func waitForRuntimeRelease(ops wslPortOps, conflicts []ports.Conflict) []ports.Conflict {
	start := ops.now()
	for ops.now().Sub(start) < ops.releaseTimeout {
		ops.sleep(ops.releaseInterval)
		conflicts = ops.scanPorts()
		if !hasConflict(conflicts, isRuntimeHeld) {
			break
		}
	}
	return conflicts
}

func nonElevatedPortError(ops wslPortOps, conflicts []ports.Conflict, needWSLShutdown bool) error {
	if needWSLShutdown {
		return fmt.Errorf(
			"failed to start scheduler service: %s\n\n"+
				"A required port is already in use (often due to WSL).\n"+
				"To resolve this, re-run 'dapr init' in an elevated (Administrator)\n"+
				"terminal (e.g. right-click → \"Run as administrator\"). When running\n"+
				"elevated, the CLI will automatically stop and restart WSL and\n"+
				"Windows networking services as part of the installation process",
			firstWSLConflict(conflicts))
	}
	// Only reserved ports remain. Stopping WinNAT releases its dynamic
	// reservations; with WinNAT not running the exclusion is persistent and
	// elevation would not help.
	if !ops.isWinNATRunning() {
		return stillUnavailableError(conflicts)
	}
	return fmt.Errorf("failed to start scheduler service: %s\n\n"+
		"Windows reserves dynamic port ranges for Hyper-V/WinNAT (see 'netsh int ipv4 show excludedportrange protocol=tcp').\n"+
		"To resolve this, re-run 'dapr init' in an elevated (Administrator)\n"+
		"terminal (e.g. right-click → \"Run as administrator\"). When running\n"+
		"elevated, the CLI will temporarily stop the Windows NAT service to release\n"+
		"the reservation; WSL is not shut down",
		conflicts[0])
}

func stillUnavailableError(conflicts []ports.Conflict) error {
	lines := make([]string, 0, len(conflicts))
	reserved := false
	for _, c := range conflicts {
		lines = append(lines, "  - "+c.String())
		reserved = reserved || c.Status == ports.Reserved
	}
	msg := "failed to start scheduler service: scheduler ports could not be freed by stopping WSL/Windows NAT:\n" +
		strings.Join(lines, "\n")
	if reserved {
		msg += "\n\nA port still reserved with the Windows NAT service stopped is in a persistent excluded port range.\n" +
			"Check 'netsh int ipv4 show excludedportrange protocol=tcp' (entries marked '*' are administered exclusions)\n" +
			"and remove it with 'netsh int ipv4 delete excludedportrange protocol=tcp startport=<port> numberofports=1'"
	}
	return fmt.Errorf("%s", msg)
}

func isRuntimeHeld(c ports.Conflict) bool {
	return c.Status == ports.InUse && c.Holder == ports.HolderContainerRuntime
}

func isOtherProcessHeld(c ports.Conflict) bool {
	return c.Status == ports.InUse && c.Holder == ports.HolderOther
}

func isUnexpectedBindError(c ports.Conflict) bool {
	return c.Status == ports.Unavailable
}

func isUnknownHeld(c ports.Conflict) bool {
	return c.Status == ports.InUse && c.Holder == ports.HolderUnknown
}

// isWSLOrUnknownHeld reports conflicts that a WSL shutdown may fix: a port in
// use by WSL, or in use by an owner that could not be determined (only
// reached while a WSL distribution is running).
func isWSLOrUnknownHeld(c ports.Conflict) bool {
	return c.Status == ports.InUse && (c.Holder == ports.HolderWSL || c.Holder == ports.HolderUnknown)
}

func firstWSLConflict(conflicts []ports.Conflict) ports.Conflict {
	if c, ok := firstConflict(conflicts, isWSLOrUnknownHeld); ok {
		return c
	}
	return conflicts[0]
}

func firstConflict(conflicts []ports.Conflict, match func(ports.Conflict) bool) (ports.Conflict, bool) {
	for _, c := range conflicts {
		if match(c) {
			return c, true
		}
	}
	return ports.Conflict{}, false
}

func hasConflict(conflicts []ports.Conflict, match func(ports.Conflict) bool) bool {
	_, ok := firstConflict(conflicts, match)
	return ok
}
