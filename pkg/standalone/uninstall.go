package standalone

import (
	"fmt"
	"strings"

	"github.com/dapr/cli/utils"
)

// Uninstall deletes all installed containers
func Uninstall(uninstallAll bool, dockerNetwork string) error {
	var failedContainers []string

	err := utils.RunCmdAndWait(
		"docker", "rm",
		"--force",
		utils.CreateContainerName(DaprPlacementContainerName, dockerNetwork))

	if err != nil {
		failedContainers = append(failedContainers, DaprPlacementContainerName)
	}

	err = utils.RunCmdAndWait(
		"docker", "rmi", "--force",
		daprDockerImageName)

	if err != nil {
		failedContainers = append(failedContainers, daprDockerImageName)
	}

	if uninstallAll {
		err = utils.RunCmdAndWait(
			"docker", "rm", "--force",
			utils.CreateContainerName(DaprRedisContainerName, dockerNetwork))
		if err != nil {
			failedContainers = append(failedContainers, DaprRedisContainerName)
		}
	}

	if len(failedContainers) == 0 {
		return nil
	}

	return fmt.Errorf("could not delete (%s)", strings.Join(failedContainers, ","))
}
