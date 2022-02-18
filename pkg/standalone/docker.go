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

func loadDocker(in io.Reader) error {
	subProcess := exec.Command("docker", "load")

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

func loadDockerIfNecessary(dockerImage string) error {
	if !isEmbedded {
		return nil
	}

	var imageFile io.Reader
	var err error
	imageFile, err = binaries.Open(path_filepath.Join("staging", "images", imageFileName(dockerImage)))
	if err != nil {
		return fmt.Errorf("fail to read docker image file %s: %v", dockerImage, err)
	}
	err = loadDocker(imageFile)
	if err != nil {
		return fmt.Errorf("fail to load docker image %s: %v", dockerImage, err)
	}

	return nil
}

// check if the container either exists and stopped or is running.
func confirmContainerIsRunningOrExists(containerName string, isRunning bool) (bool, error) {
	// e.g. docker ps --filter name=dapr_redis --filter status=running --format {{.Names}}

	args := []string{"ps", "--all", "--filter", "name=" + containerName}

	if isRunning {
		args = append(args, "--filter", "status=running")
	}

	args = append(args, "--format", "{{.Names}}")
	response, err := utils.RunCmdAndWait("docker", args...)
	response = strings.TrimSuffix(response, "\n")

	// If 'docker ps' failed due to some reason
	if err != nil {
		return false, fmt.Errorf("unable to confirm whether %s is running or exists. error\n%v", containerName, err.Error())
	}
	// 'docker ps' worked fine, but the response did not have the container name
	if response == "" || response != containerName {
		if isRunning {
			return false, fmt.Errorf("container %s is not running", containerName)
		}
		return false, nil
	}

	return true, nil
}

func isContainerRunError(err error) bool {
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode := exitError.ExitCode()
		return exitCode == 125
	}
	return false
}

func parseDockerError(component string, err error) error {
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
