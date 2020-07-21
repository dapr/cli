package standalone

import (
	"errors"
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/print"
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
		print.WarningStatusEvent(os.Stdout, "WARNING: %s container does not exist", containerName)
		return containerErrs
	}
	print.InfoStatusEvent(os.Stdout, "Removing container: %s", containerName)
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
	defaultDaprPath := defaultDaprDirPath()
	_, err := os.Stat(defaultDaprPath)
	if os.IsNotExist(err) {
		print.WarningStatusEvent(os.Stdout, "WARNING: %s default Dapr folder does not exist", defaultDaprPath)
		return defaultDaprPath, nil
	}
	print.InfoStatusEvent(os.Stdout, "Removing folder: %s", defaultDaprPath)
	err = os.RemoveAll(defaultDaprPath)

	return defaultDaprPath, err
}

func removeFileIfExists(binaryFilePath string) error {
	_, err := os.Stat(binaryFilePath)
	if os.IsNotExist(err) {
		return nil
	}
	print.InfoStatusEvent(os.Stdout, "Removing file: %s", binaryFilePath)
	err = os.Remove(binaryFilePath)

	return err
}

// Uninstall reverts all changes made by init. Deletes all installed containers, removes default dapr folder,
// removes the installed binary and unsets env variables.
func Uninstall(uninstallAll bool, dockerNetwork string) error {
	var containerErrs []error
	var err error
	var path string

	daprdFilePath := binaryFilePath(daprRuntimeFilePrefix)
	err = removeFileIfExists(daprdFilePath)
	if err != nil {
		print.WarningStatusEvent(os.Stdout, "WARNING: could not delete binary file: %s", daprdFilePath)
	}

	placementFilePath := binaryFilePath(placementServiceFilePrefix)
	_, placementErr := os.Stat(placementFilePath) // check if the placement binary exists
	uninstallPlacementContainer := os.IsNotExist(placementErr)
	err = removeFileIfExists(placementFilePath)
	if err != nil {
		print.WarningStatusEvent(os.Stdout, "WARNING: could not delete binary file: %s", placementFilePath)
	}

	dockerInstalled := false
	dockerInstalled = utils.IsDockerInstalled()
	if dockerInstalled {
		containerErrs = removeContainers(uninstallPlacementContainer, uninstallAll, dockerNetwork)
	}

	path, err = removeDefaultDaprDir(uninstallAll)
	if err != nil {
		print.WarningStatusEvent(os.Stdout, "WARNING: could not delete default dapr folder: %s", path)
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
