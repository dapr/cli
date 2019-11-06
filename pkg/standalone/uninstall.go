package standalone

import (
	"errors"

	"github.com/dapr/cli/utils"
)

func Uninstall(uninstallAll bool, dockerNetwork string) error {
	err := utils.RunCmdAndWait(
		"docker", "rm",
		"--force",
		utils.CreateContainerName(DaprPlacementContainerName, dockerNetwork))

	errorMessage := ""
	if err != nil {
		errorMessage += "Could not delete Dapr Placement Container - it may not have been running "
	}

	if uninstallAll {
		err = utils.RunCmdAndWait(
			"docker", "rm",
			"--force",
			utils.CreateContainerName(DaprRedisContainerName, dockerNetwork))
		if err != nil {
			errorMessage += "Could not delete Redis Container - it may not have been running"
		}
	}

	if errorMessage != "" {
		return errors.New(errorMessage)
	}
	return nil
}
