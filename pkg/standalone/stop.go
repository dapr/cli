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
		return fmt.Errorf("error: %w", err)
	}

	for _, a := range apps {
		if a.AppID == appID {
			pid := fmt.Sprintf("%v", a.PID)

			_, err := utils.RunCmdAndWait("kill", pid)

			return fmt.Errorf("error: %w", err)
		}
	}

	return fmt.Errorf("couldn't find app id %s", appID)
}
