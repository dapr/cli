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
	"strconv"
	"syscall"
	"time"

	"github.com/dapr/cli/utils"
	"github.com/kolesnikovae/go-winjob"
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
			return disposeJobHandle(a.CliPID)
		}
	}
	return fmt.Errorf("couldn't find apps with run file %q", runTemplatePath)
}

func disposeJobHandle(cliPID int) error {
	jbobj, err := winjob.Open(strconv.Itoa(cliPID) + "-" + utils.WindowsDaprAppProcJobName)
	if err != nil {
		return fmt.Errorf("error opening job object: %w", err)
	}
	err = jbobj.TerminateWithExitCode(0)
	if err != nil {
		return fmt.Errorf("error terminating job object: %w", err)
	}
	time.Sleep(5 * time.Second)
	return handleEvent(cliPID)
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
