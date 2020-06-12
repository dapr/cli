package standalone

import (
	"errors"
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/rundata"
	"github.com/dapr/cli/utils"
)

func removeContainers(uninstallAll bool, dockerNetwork string) []error {
	var containerErrs []error

	_, err := utils.RunCmdAndWait(
		"docker", "rm",
		"--force",
		utils.CreateContainerName(DaprPlacementContainerName, dockerNetwork))

	if err != nil {
		containerErrs = append(
			containerErrs,
			fmt.Errorf("could not remove %s container: %s", DaprPlacementContainerName, err))
	}

	_, err = utils.RunCmdAndWait(
		"docker", "rmi",
		"--force",
		daprDockerImageName)

	if err != nil {
		containerErrs = append(
			containerErrs,
			fmt.Errorf("could not remove %s image: %s", daprDockerImageName, err))
	}

	// Add the zipkin container in uninstall all block
	if uninstallAll {
		fmt.Println("trying to remove redis container: ", DaprRedisContainerName)
		_, err = utils.RunCmdAndWait(
			"docker", "rm",
			"--force",
			utils.CreateContainerName(DaprRedisContainerName, dockerNetwork))
		if err != nil {
			containerErrs = append(
				containerErrs,
				fmt.Errorf("could not remove %s container: %s", DaprRedisContainerName, err))
		}
		fmt.Println("trying to remove zipkin container: ", DaprZipkinContainerName)
		_, err = utils.RunCmdAndWait(
			"docker", "rm",
			"--force",
			utils.CreateContainerName(DaprZipkinContainerName, dockerNetwork))
		if err != nil {
			containerErrs = append(
				containerErrs,
				fmt.Errorf("could not remove %s container: %s", DaprZipkinContainerName, err))
		}
	}

	return containerErrs
}

func removeDefaultDaprDir(uninstallAll bool) (string, error) {
	if !uninstallAll {
		return "", nil
	}
	defaultDaprPath := GetDefaultFolderPath(defaultDaprDirName)
	fmt.Println("removing .dapr folder: ", defaultDaprPath)
	err := os.RemoveAll(defaultDaprPath)

	return defaultDaprPath, err
}

// Uninstall reverts all changes made by init. Deletes all installed containers, removes default components, config folder, unsets env variables.
func Uninstall(uninstallAll bool, dockerNetwork string) error {
	var containerErrs []error

	dockerInstalled := utils.IsDockerInstalled()
	if dockerInstalled {
		containerErrs = removeContainers(uninstallAll, dockerNetwork)
	}

	err := rundata.DeleteRunDataFile()
	if err != nil {
		fmt.Println("WARNING: could not delete run data file")
	}

	daprPath, err := removeDefaultDaprDir(uninstallAll)
	if err != nil {
		fmt.Println("WARNING: could not delete default dapr folder: ", daprPath)
	}

	err = errors.New("uninstall failed")
	if !dockerInstalled {
		return fmt.Errorf("%w \n could not connect to Docker. Docker may not be installed or running", err)
	}

	if len(containerErrs) == 0 {
		return nil
	}

	for _, e := range containerErrs {
		err = fmt.Errorf("%w \n %s", err, e)
	}
	return err
}
