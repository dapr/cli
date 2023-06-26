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
	"fmt"
	"os/exec"
	"syscall"

	ps "github.com/mitchellh/go-ps"
	process "github.com/shirou/gopsutil/process"
	"golang.org/x/sys/windows"
)

// Stop terminates the application process.
func Stop(appID string, cliPIDToNoOfApps map[int]int, apps []ListOutput) error {
	for _, a := range apps {
		if a.AppID == appID {
			return handleEvent(a.CliPID)
		}
	}
	return fmt.Errorf("couldn't find app id %s", appID)
}

// StopAppsWithRunFile terminates the daprd and application processes with the given run file.
func StopAppsWithRunFile(runTemplatePath string) error {
	apps, err := List()
	if err != nil {
		return err
	}
	for _, a := range apps {
		if a.RunTemplatePath == runTemplatePath {
			return killProcessAndChildren(a.CliPID)
		}
	}
	return fmt.Errorf("couldn't find apps with run file %q", runTemplatePath)
}

func killProcessAndChildren(cliPID int) error {
	processes, err := ps.Processes()
	if err != nil {
		return err
	}

	for _, p := range processes {
		if p.Pid() == cliPID {
			proc, err := process.NewProcess(int32(p.Pid()))
			if err != nil {
				return err
			}
			processName, err := proc.Name()
			if err != nil {
				return err
			}
			killCMD := exec.Command("taskkill", "/im", processName, "/T", "/F")
			err = killCMD.Run()
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("process not found")
}

func handleEvent(cliPID int) error {
	eventName, _ := syscall.UTF16FromString(fmt.Sprintf("dapr_cli_%v", cliPID))
	eventHandle, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, &eventName[0])
	if err != nil {
		return err
	}

	err = windows.SetEvent(eventHandle)
	return err
}
