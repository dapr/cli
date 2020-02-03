// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"fmt"
	"runtime"

	"github.com/dapr/cli/utils"
)

// Stop terminates the application process.
func Stop(appID string) error {
	apps, err := List()
	if err != nil {
		return err
	}

	for _, a := range apps {
		if a.AppID == appID {
			pid := fmt.Sprintf("%v", a.PID)

			var err error
			if runtime.GOOS == "windows" {
				_, err = utils.RunCmdAndWait("taskkill", "/F", "/PID", pid)
			} else {
				_, err = utils.RunCmdAndWait("kill", pid)
			}

			return err
		}
	}

	return fmt.Errorf("couldn't find app id %s", appID)
}
