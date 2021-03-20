// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

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
			eventName, _ := syscall.UTF16FromString(fmt.Sprintf("dapr_cli_%v", a.PID))
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


