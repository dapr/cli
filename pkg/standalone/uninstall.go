/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package standalone

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

func removeContainers(uninstallPlacementContainer, uninstallAll bool, dockerNetwork, runtimeCmd string) []error {
	var containerErrs []error

	if uninstallPlacementContainer {
		containerErrs = removeDockerContainer(containerErrs, DaprPlacementContainerName, dockerNetwork, runtimeCmd)
	}

	if uninstallAll {
		containerErrs = removeDockerContainer(containerErrs, DaprRedisContainerName, dockerNetwork, runtimeCmd)
		containerErrs = removeDockerContainer(containerErrs, DaprZipkinContainerName, dockerNetwork, runtimeCmd)
	}

	return containerErrs
}

func removeDockerContainer(containerErrs []error, containerName, network, runtimeCmd string) []error {
	container := utils.CreateContainerName(containerName, network)
	exists, _ := confirmContainerIsRunningOrExists(container, false, runtimeCmd)
	if !exists {
		print.WarningStatusEvent(os.Stdout, "WARNING: %s container does not exist", container)
		return containerErrs
	}
	print.InfoStatusEvent(os.Stdout, "Removing container: %s", container)
	_, err := utils.RunCmdAndWait(
		runtimeCmd, "rm",
		"--force",
		container)
	if err != nil {
		containerErrs = append(
			containerErrs,
			fmt.Errorf("could not remove %s container: %w", container, err))
	}
	return containerErrs
}

func removeDir(dirPath string) error {
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		print.WarningStatusEvent(os.Stdout, "WARNING: %s does not exist", dirPath)
		return nil
	}
	print.InfoStatusEvent(os.Stdout, "Removing directory: %s", dirPath)
	err = os.RemoveAll(dirPath)
	return err
}

// Uninstall reverts all changes made by init. Deletes all installed containers, removes default dapr folder,
// removes the installed binary and unsets env variables.
func Uninstall(uninstallAll bool, dockerNetwork string, containerRuntime string, inputInstallPath string) error {
	var containerErrs []error
	inputInstallPath = strings.TrimSpace(inputInstallPath)
	installDir, err := GetDaprRuntimePath(inputInstallPath)
	if err != nil {
		return err
	}
	daprBinDir := getDaprBinPath(installDir)

	placementFilePath := binaryFilePathWithDir(daprBinDir, placementServiceFilePrefix)
	_, placementErr := os.Stat(placementFilePath) // check if the placement binary exists.
	uninstallPlacementContainer := errors.Is(placementErr, fs.ErrNotExist)
	// Remove .dapr/bin.
	err = removeDir(daprBinDir)
	if err != nil {
		print.WarningStatusEvent(os.Stdout, "WARNING: could not delete dapr bin dir: %s", daprBinDir)
	}

	containerRuntime = strings.TrimSpace(containerRuntime)
	runtimeCmd := utils.GetContainerRuntimeCmd(containerRuntime)
	containerRuntimeAvailable := false
	containerRuntimeAvailable = utils.IsContainerRuntimeInstalled(containerRuntime)
	if containerRuntimeAvailable {
		containerErrs = removeContainers(uninstallPlacementContainer, uninstallAll, dockerNetwork, runtimeCmd)
	}

	if uninstallAll {
		err = removeDir(installDir)
		if err != nil {
			print.WarningStatusEvent(os.Stdout, "WARNING: could not delete dapr dir %s: %s", installDir, err)
		}
	}

	err = errors.New("uninstall failed")

	if len(containerErrs) == 0 {
		return nil
	}

	// TODO move to use errors.Join once we move to go 1.20.
	for _, e := range containerErrs {
		err = fmt.Errorf("%w \n %w", err, e)
	}
	return err
}
