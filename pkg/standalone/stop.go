// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"fmt"
	"runtime"
)

func Stop(appID string) error {
	apps, err := List()
	if err != nil {
		return err
	}

	for _, a := range apps {
		if a.AppID == appID {
			pid := fmt.Sprintf("%v", a.PID)
			if runtime.GOOS == "windows" {
				err := runCmd("taskkill", "/F", "/PID", pid)
				return err
			} else {
				err := runCmd("kill", pid)
				return err
			}
		}
	}

	return fmt.Errorf("couldn't find app id %s", appID)
}
