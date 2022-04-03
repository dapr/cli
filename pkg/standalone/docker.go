/*
Copyright 2022 The Dapr Authors
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
	"fmt"
	"io"
	"os"
	"os/exec"
	path_filepath "path/filepath"
	"strings"

	"github.com/dapr/cli/utils"
)

func runDockerLoad(in io.Reader) error {
	runtimeCmd := utils.GetContainerRuntimeCmd()
	subProcess := exec.Command(runtimeCmd, "load")

	stdin, err := subProcess.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()

	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr

	if err = subProcess.Start(); err != nil {
		return err
	}

	if _, err = io.Copy(stdin, in); err != nil {
		return err
	}

	stdin.Close()

	if err = subProcess.Wait(); err != nil {
		return err
	}

	return nil
}

func loadDocker(dir string, dockerImageFileName string) error {
	var imageFile io.Reader
	var err error
	imageFile, err = os.Open(path_filepath.Join(dir, dockerImageFileName))
	if err != nil {
		return fmt.Errorf("fail to read docker image file %s: %w", dockerImageFileName, err)
	}
	err = runDockerLoad(imageFile)
	if err != nil {
		return fmt.Errorf("fail to load docker image from file %s: %w", dockerImageFileName, err)
	}

	return nil
}

// check if the container either exists and stopped or is running.
func confirmContainerIsRunningOrExists(containerName string, isRunning bool) (bool, error) {
	// e.g. docker ps --filter name=dapr_redis --filter status=running --format {{.Names}}.

	args := []string{"ps", "--all", "--filter", "name=" + containerName}

	if isRunning {
		args = append(args, "--filter", "status=running")
	}

	runtimeCmd := utils.GetContainerRuntimeCmd()
	args = append(args, "--format", "{{.Names}}")
	response, err := utils.RunCmdAndWait(runtimeCmd, args...)
	response = strings.TrimSuffix(response, "\n")

	// If 'docker ps' failed due to some reason.
	if err != nil {
		//nolint
		return false, fmt.Errorf("unable to confirm whether %s is running or exists. error\n%v", containerName, err.Error())
	}
	// 'docker ps' worked fine, but the response did not have the container name.
	if response == "" || response != containerName {
		if isRunning {
			return false, fmt.Errorf("container %s is not running", containerName)
		}
		return false, nil
	}

	return true, nil
}

func isContainerRunError(err error) bool {
	//nolint
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode := exitError.ExitCode()
		return exitCode == 125
	}
	return false
}

func parseDockerError(component string, err error) error {
	//nolint
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode := exitError.ExitCode()
		if exitCode == 125 { // see https://github.com/moby/moby/pull/14012
			return fmt.Errorf("failed to launch %s. Is it already running?", component)
		}
		if exitCode == 127 {
			return fmt.Errorf("failed to launch %s. Make sure Docker is installed and running", component)
		}
	}
	return err
}

func tryPullImage(imageName string) bool {
	runtimeCmd := utils.GetContainerRuntimeCmd()
	args := []string{
		"pull",
		imageName,
	}
	_, err := utils.RunCmdAndWait(runtimeCmd, args...)
	return err == nil
}
