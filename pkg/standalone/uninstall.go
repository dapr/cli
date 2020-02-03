package standalone

import (
	"errors"
	"fmt"

	"github.com/dapr/cli/utils"
)

func Uninstall(uninstallAll bool, dockerNetwork string) error {
	_, err := utils.RunCmdAndWait(
		"docker", "rm",
		"--force",
		utils.CreateContainerName(DaprPlacementContainerName, dockerNetwork))

	errorMessage := ""
	if err != nil {
		errorMessage += "Could not delete Dapr Placement Container - it may not have been running "
	}

	_, err = utils.RunCmdAndWait(
		"docker", "rmi",
		"--force",
		daprDockerImageName)

	errorMessage = ""
	if err != nil {
		errorMessage += fmt.Sprintf("Could not delete image %s - it may not be present on the host", daprDockerImageName)
	}

	if uninstallAll {
		_, err = utils.RunCmdAndWait(
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
