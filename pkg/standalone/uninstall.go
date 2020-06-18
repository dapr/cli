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
	var err error

	containerErrs = removeDockerContainer(containerErrs, DaprPlacementContainerName, dockerNetwork)

	_, err = utils.RunCmdAndWait(
		"docker", "rmi",
		"--force",
		daprDockerImageName)

	if err != nil {
		containerErrs = append(
			containerErrs,
			fmt.Errorf("could not remove %s image: %s", daprDockerImageName, err))
	}

	if uninstallAll {
		containerErrs = removeDockerContainer(containerErrs, DaprRedisContainerName, dockerNetwork)
		containerErrs = removeDockerContainer(containerErrs, DaprZipkinContainerName, dockerNetwork)
	}

	return containerErrs
}

func removeDockerContainer(containerErrs []error, containerName, network string) []error {
	fmt.Println("trying to remove container: ", containerName)
	_, err := utils.RunCmdAndWait(
		"docker", "rm",
		"--force",
		utils.CreateContainerName(containerName, network))

	if err != nil {
		containerErrs = append(
			containerErrs,
			fmt.Errorf("could not remove %s container: %s", containerName, err))
	}
	return containerErrs
}

func removeDefaultDaprDir(uninstallAll bool) (string, error) {
	if !uninstallAll {
		return "", nil
	}
	defaultDaprPath := DefaultFolderPath(defaultDaprDirName)
	fmt.Println("removing folder: ", defaultDaprPath)
	err := os.RemoveAll(defaultDaprPath)

	return defaultDaprPath, err
}

// Uninstall reverts all changes made by init. Deletes all installed containers, removes default dapr folder unsets env variables.
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
