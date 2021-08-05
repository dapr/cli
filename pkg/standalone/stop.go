// +build !windows

// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"fmt"

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
			pid := fmt.Sprintf("%v", a.CliPID)
			if pid == "0" {
				pid = fmt.Sprintf("%v", a.DaprdPID)
			}

			_, err := utils.RunCmdAndWait("kill", pid)

			return err
		}
	}

	return fmt.Errorf("couldn't find app id %s", appID)
}
