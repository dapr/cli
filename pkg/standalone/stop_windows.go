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
	"syscall"

	"golang.org/x/sys/windows"
)

// Stop terminates the application process.
func Stop(appID string) error {
	apps, err := List()
	if err != nil {
		return err
	}

	for _, a := range apps {
		if a.AppID == appID {
			var pid string

			// Unlike Linux/Mac, we can't just send sigterm. In order for 'dapr shutdown'
			// to be able to receive the signal gracefully, we must send a named event in Windows.
			//
			// Kill the Daprd process if Daprd was started without CLI, otherwise
			// kill the CLI process which also kills the associated Daprd process.
			// TODO: who handles dapr_daprd_ named events?
			if a.CliPID == 0 {
				pid = fmt.Sprintf("dapr_daprd_%v", a.DaprdPID)
			} else {
				pid = fmt.Sprintf("dapr_cli_%v", a.CliPID)
			}
			eventName, _ := syscall.UTF16FromString(pid)

			eventHandle, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, &eventName[0])
			if err != nil {
				return err
			}

			err = windows.SetEvent(eventHandle)
			return err
		}
	}

	return fmt.Errorf("couldn't find app id %s", appID)
}
