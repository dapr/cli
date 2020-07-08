package standalone

import (
	"errors"
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/rundata"
	"github.com/dapr/cli/utils"
)

func removeContainers(uninstallPlacementContainer, uninstallAll bool, dockerNetwork string) []error {
	var containerErrs []error
	var err error

	if uninstallPlacementContainer {
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
	}

	if uninstallAll {
		containerErrs = removeDockerContainer(containerErrs, DaprRedisContainerName, dockerNetwork)
		containerErrs = removeDockerContainer(containerErrs, DaprZipkinContainerName, dockerNetwork)
	}

	return containerErrs
}

func removeDockerContainer(containerErrs []error, containerName, network string) []error {
	exists, _ := confirmContainerIsRunningOrExists(containerName, false)
	if !exists {
		fmt.Printf("WARNING: %s container does not exist\n", containerName)
		return containerErrs
	}
	fmt.Println("removing container: ", containerName)
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
	defaultDaprPath := defaultFolderPath(defaultDaprDirName)
	fmt.Println("removing folder: ", defaultDaprPath)
	err := os.RemoveAll(defaultDaprPath)

	return defaultDaprPath, err
}

func removeInstalledBinaries(binaryFilePrefix, installLocation string) (string, error) {
	binaryPath := binaryFilePath(binaryFilePrefix, installLocation)
	_, err := os.Stat(binaryPath)
	if os.IsNotExist(err) {
		return binaryPath, nil
	}
	fmt.Println("removing binary: ", binaryPath)
	err = os.Remove(binaryPath)

	return binaryPath, err
}

// Uninstall reverts all changes made by init. Deletes all installed containers, removes default dapr folder,
// removes the installed binary and unsets env variables.
func Uninstall(uninstallAll bool, installLocation, dockerNetwork string) error {
	var containerErrs []error
	var err error
	var path string

	path, err = removeInstalledBinaries(daprRuntimeFilePrefix, installLocation)
	if err != nil {
		fmt.Println("WARNING: could not delete binary file: ", path)
	}

	placementFilePath := binaryFilePath(placementServiceFilePrefix, installLocation)
	_, placementErr := os.Stat(placementFilePath) // check if the placement binary exists
	uninstallPlacementContainer := os.IsNotExist(placementErr)
	path, err = removeInstalledBinaries(placementServiceFilePrefix, installLocation)
	if err != nil {
		fmt.Println("WARNING: could not delete binary file: ", path)
	}

	dockerInstalled := false
	dockerInstalled = utils.IsDockerInstalled()
	if dockerInstalled {
		containerErrs = removeContainers(uninstallPlacementContainer, uninstallAll, dockerNetwork)
	}

	err = rundata.DeleteRunDataFile()
	if err != nil {
		fmt.Println("WARNING: could not delete run data file")
	}

	path, err = removeDefaultDaprDir(uninstallAll)
	if err != nil {
		fmt.Println("WARNING: could not delete default dapr folder: ", path)
	}

	err = errors.New("uninstall failed")
	if uninstallPlacementContainer && !dockerInstalled {
		// if placement binary did not exist before trying to delete it and not able to connect to docker.
		return fmt.Errorf("%w \ncould not delete placement service. Either the placement binary is not found, or Docker may not be installed or running", err)
	}

	if len(containerErrs) == 0 {
		return nil
	}

	for _, e := range containerErrs {
		err = fmt.Errorf("%w \n %s", err, e)
	}
	return err
}
