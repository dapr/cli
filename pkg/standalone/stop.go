//go:build !windows
// +build !windows

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
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/dapr/cli/utils"
)

// Stop terminates the application process.
func Stop(appID string, cliPIDToNoOfApps map[int]int, apps []ListOutput, timeout time.Duration) error {
	for _, a := range apps {
		if a.AppID == appID {
			var pid int
			// Kill the Daprd process if Daprd was started without CLI, otherwise
			// kill the CLI process which also kills the associated Daprd process.
			if a.CliPID == 0 || cliPIDToNoOfApps[a.CliPID] > 1 {
				pid = a.DaprdPID //nolint: perfsprint
				cliPIDToNoOfApps[a.CliPID]--
			} else {
				pid = a.CliPID //nolint: perfsprint
			}

			_, err := utils.RunCmdAndWait("kill", fmt.Sprintf("%v", pid)) //nolint:perfsprint
			if err != nil {
				return err
			}

			return waitForProccessToExit(pid, timeout)
		}
	}
	return fmt.Errorf("couldn't find app id %s", appID)
}

// StopAppsWithRunFile terminates the daprd and application processes with the given run file.
func StopAppsWithRunFile(runTemplatePath string, timeout time.Duration) error {
	apps, err := List()
	if err != nil {
		return err
	}
	for _, a := range apps {
		if a.RunTemplatePath == runTemplatePath {
			// Get the process group id of the CLI process.
			pgid, err := syscall.Getpgid(a.CliPID)
			if err != nil {
				// Fall back to cliPID if pgid is not available.
				_, err = utils.RunCmdAndWait("kill", fmt.Sprintf("%v", a.CliPID)) //nolint:perfsprint
				if err != nil {
					return err
				}
				return waitForProccessToExit(a.CliPID, timeout)
			}
			// Kill the whole process group.
			err = syscall.Kill(-pgid, syscall.SIGINT)
			if err != nil {
				return err
			}
			return waitForProccessToExit(-pgid, timeout)
		}
	}
	return fmt.Errorf("couldn't find apps with run file %q", runTemplatePath)
}

func waitForProccessToExit(pid int, timeout time.Duration) error {
	if timeout == 0 {
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil //nolint:nilerr
	}

	ticker := time.NewTicker(time.Second)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if err := proc.Signal(syscall.Signal(0)); err != nil && !errors.Is(err, os.ErrProcessDone) {
				return err
			} else if err != nil {
				return nil //nolint:nilerr
			}
		case <-timer.C:
			proc.Signal(syscall.SIGKILL)
			return nil
		}
	}
}
