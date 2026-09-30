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
	"time"

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
)

// wslPortOps holds the operations used to free the scheduler's host ports on
// Windows when WSL2 is holding them. Fields are injectable for testing.
type wslPortOps struct {
	elevated        func() bool
	checkPorts      func() error
	shutdownWSL     func() error
	isWinNATRunning func() bool
	stopWinNAT      func() error
	startWinNAT     func() error
	startWSL        func()
	waitForRuntime  func() error
}

func defaultWSLPortOps(runtimeCmd string, grpcPort int) wslPortOps {
	return wslPortOps{
		elevated:        isWindowsElevated,
		checkPorts:      func() error { return checkSchedulerPorts(grpcPort) },
		shutdownWSL:     shutdownWSL,
		isWinNATRunning: isWinNATRunning,
		stopWinNAT:      stopWinNAT,
		startWinNAT:     startWinNAT,
		startWSL:        startWSLBackground,
		waitForRuntime: func() error {
			return waitForContainerRuntime(runtimeCmd, containerRuntimeRecoveryTimeout, containerRuntimeRecoveryInterval)
		},
	}
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
// one of them is in use.
//
// When a port is busy and the process is elevated, it shuts down WSL and stops
// WinNAT (if running) and waits for the container runtime to be reachable
// again. On failure WinNAT/WSL are restored before
// returning. On success the caller owns the returned restore func, which is
// nil when nothing was stopped.
func prepareSchedulerHostPorts(ops wslPortOps) (restore func(), err error) {
	portErr := ops.checkPorts()
	if portErr == nil {
		return nil, nil
	}

	if !ops.elevated() {
		return nil, fmt.Errorf(
			"failed to start scheduler service: %v\n\n"+
				"A required port is already in use (often due to WSL).\n"+
				"To resolve this, re-run 'dapr init' in an elevated (Administrator)\n"+
				"terminal (e.g. right-click → \"Run as administrator\"). When running\n"+
				"elevated, the CLI will automatically stop and restart WSL and\n"+
				"Windows networking services as part of the installation process",
			portErr)
	}

	print.InfoStatusEvent(os.Stdout, "Scheduler port in use (%v). Temporarily shutting down WSL to free it...", portErr)
	if wslErr := ops.shutdownWSL(); wslErr != nil {
		print.WarningStatusEvent(os.Stdout, "Failed to shut down WSL: %v. Continuing...", wslErr)
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

	restore = func() {
		if winNATStopped {
			if startErr := ops.startWinNAT(); startErr != nil {
				print.WarningStatusEvent(os.Stdout, "Failed to restart Windows NAT service: %v", startErr)
			}
		}
		print.InfoStatusEvent(os.Stdout, "Restarting WSL...")
		ops.startWSL()
	}

	print.InfoStatusEvent(os.Stdout, "Waiting for the container runtime to become available after WSL shutdown...")
	if waitErr := ops.waitForRuntime(); waitErr != nil {
		restore()
		return nil, fmt.Errorf(
			"failed to start scheduler service: container runtime did not become available after shutting down WSL: %w\n\n"+
				"If you are using Docker Desktop, wait until it reports that the engine is running and re-run 'dapr init'",
			waitErr)
	}

	return restore, nil
}
