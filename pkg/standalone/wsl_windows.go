//go:build windows

/*
Copyright 2021 The Dapr Authors
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
	"os/exec"

	"golang.org/x/sys/windows"

	"github.com/dapr/cli/utils"
)

// isWindowsElevated returns true if the current process is running with
// elevated (Administrator) privileges.
func isWindowsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// isWSLAvailable returns true if the wsl executable is available in PATH.
func isWSLAvailable() bool {
	_, err := exec.LookPath("wsl")
	return err == nil
}

// shutdownWSL runs `wsl --shutdown` to terminate the WSL2 VM and free any
// ports it holds.
func shutdownWSL() error {
	_, err := utils.RunCmdAndWait("wsl", "--shutdown")
	return err
}

// stopWinNAT stops the Windows NAT driver service (WinNat) so that Docker
// can re-acquire port bindings that WinNAT was caching.
func stopWinNAT() error {
	_, err := utils.RunCmdAndWait("net", "stop", "winnat")
	return err
}

// startWinNAT starts the Windows NAT driver service after the scheduler
// container has been created.
func startWinNAT() error {
	_, err := utils.RunCmdAndWait("net", "start", "winnat")
	return err
}

// startWSLBackground starts WSL in the background to re-initialize WSL2
// networking after a wsl --shutdown. We run a no-op command so the session
// exits immediately once WSL services are up, then wait in a goroutine to
// clean up the process handle.
func startWSLBackground() {
	cmd := exec.Command("wsl", "--exec", "echo")
	if err := cmd.Start(); err != nil {
		return
	}
	go func() { _ = cmd.Wait() }()
}
