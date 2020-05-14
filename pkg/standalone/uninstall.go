package standalone

import (
	"errors"
	"fmt"

	"github.com/dapr/cli/pkg/rundata"
	"github.com/dapr/cli/utils"
)

// Uninstall deletes all installed containers
func Uninstall(uninstallAll bool, dockerNetwork string) error {
	var errs []error

	_, err := utils.RunCmdAndWait(
		"docker", "rm",
		"--force",
		utils.CreateContainerName(DaprPlacementContainerName, dockerNetwork))

	if err != nil {
		errs = append(
			errs,
			fmt.Errorf("could not remove container %s container: %s", DaprPlacementContainerName, err))
	}

	_, err = utils.RunCmdAndWait(
		"docker", "rmi",
		"--force",
		daprDockerImageName)

	if err != nil {
		errs = append(errs, fmt.Errorf("could not remove container %s container: %s", daprDockerImageName, err))
	}

	if uninstallAll {
		_, err = utils.RunCmdAndWait(
			"docker", "rm",
			"--force",
			utils.CreateContainerName(DaprRedisContainerName, dockerNetwork))
		if err != nil {
			errs = append(errs, fmt.Errorf("could not remove %s container: %s", DaprRedisContainerName, err))
		}
	}

	err = rundata.DeleteRunDataFile()
	if err != nil {
		fmt.Println("WARNING: could not delete run data file")
	}

	if len(errs) == 0 {
		return nil
	}

	err = errors.New("uninstall failed")
	for _, e := range errs {
		err = fmt.Errorf("%w; %s", err, e)
	}
	return err
}
