package standalone

import (
	"errors"

	"github.com/dapr/cli/utils"
)

func Uninstall(uninstallAll bool) error {
	err := utils.RunCmdAndWait(
		"docker", "rm",
		"--force",
		DaprPlacementContainerName)

	errorMessage := ""
	if err != nil {
		errorMessage += "Could not delete Dapr Placement Container - it may not have been running "
	}

	if uninstallAll {
		err = utils.RunCmdAndWait(
			"docker", "rm",
			"--force",
			DaprRedisContainerName)
		if err != nil {
			errorMessage += "Could not delete Redis Container - it may not have been running"
		}
	}

	if errorMessage != "" {
		return errors.New(errorMessage)
	}
	return nil
}
