// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	path_filepath "path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/fatih/color"

	"github.com/briandowns/spinner"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

const (
	daprGitHubOrg                     = "dapr"
	daprGitHubRepo                    = "dapr"
	daprDockerImageName               = "daprio/dapr"
	daprRuntimeFilePrefix             = "daprd"
	daprWindowsOS                     = "windows"
	daprLatestVersion                 = "latest"
	daprDefaultLinuxAndMacInstallPath = "/usr/local/bin"
	daprDefaultWindowsInstallPath     = "c:\\dapr"

	// DaprPlacementContainerName is the container name of placement service
	DaprPlacementContainerName = "dapr_placement"
	// DaprRedisContainerName is the container name of redis
	DaprRedisContainerName = "dapr_redis"
)

// Init installs Dapr on a local machine using the supplied runtimeVersion.
func Init(runtimeVersion string, dockerNetwork string, installLocation string) error {
	dockerInstalled := isDockerInstalled()
	if !dockerInstalled {
		return errors.New("could not connect to Docker. Docker may not be installed or running")
	}

	dir, err := getDaprDir()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errorChan := make(chan error)

	initSteps := []func(*sync.WaitGroup, chan<- error, string, string, string, string){}
	initSteps = append(initSteps, installDaprBinary, runPlacementService, runRedis)

	wg.Add(len(initSteps))

	msg := "Downloading binaries and setting up components..."
	var s *spinner.Spinner
	if runtime.GOOS == daprWindowsOS {
		print.InfoStatusEvent(os.Stdout, msg)
	} else {
		s = spinner.New(spinner.CharSets[0], 100*time.Millisecond)
		s.Writer = os.Stdout
		s.Color("cyan")
		s.Suffix = fmt.Sprintf("  %s", msg)
		s.Start()
	}

	for _, step := range initSteps {
		go step(&wg, errorChan, dir, runtimeVersion, dockerNetwork, installLocation)
	}

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	for err := range errorChan {
		if err != nil {
			if s != nil {
				s.Stop()
			}
			return err
		}
	}

	if s != nil {
		s.Stop()
		err = confirmContainerIsRunning(DaprRedisContainerName)
		if err != nil {
			return err
		}
		print.SuccessStatusEvent(os.Stdout, msg)
	}

	return nil
}

func isDockerInstalled() bool {
	cli, err := client.NewEnvClient()
	if err != nil {
		return false
	}
	_, err = cli.Ping(context.Background())
	return err == nil
}

func getDaprDir() (string, error) {
	p := ""

	if runtime.GOOS == daprWindowsOS {
		p = path_filepath.FromSlash("c:/dapr")
	} else {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		p = path.Join(usr.HomeDir, ".dapr")
	}

	err := os.MkdirAll(p, 0700)
	if err != nil {
		return "", err
	}

	return p, nil
}

// installLocation is not used, but it is present because it's required to fit the initSteps func above.
// If the number of args increases more, we may consider passing in a struct instead of individual args.
func runRedis(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string, installLocation string) {
	defer wg.Done()

	args := []string{
		"run",
		"--name", utils.CreateContainerName(DaprRedisContainerName, dockerNetwork),
		"--restart", "always",
		"-d",
	}

	if dockerNetwork != "" {
		args = append(
			args,
			"--network", dockerNetwork,
			"--network-alias", DaprRedisContainerName)
	} else {
		args = append(
			args,
			"-p", "6379:6379")
	}

	args = append(args, "redis")
	_, err := utils.RunCmdAndWait("docker", args...)

	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseDockerError("Redis state store", err)
		} else {
			errorChan <- fmt.Errorf("docker %s failed with: %v", args, err)
		}
		return
	}
	errorChan <- nil
}

func confirmContainerIsRunning(containerName string) error {

	// e.g. docker ps --filter name=dapr_redis --filter status=running --format {{.Names}}

	args := []string{"ps", "--filter", "name=" + containerName, "--filter", "status=running", "--format", "{{.Names}}"}
	response, err := utils.RunCmdAndWait("docker", args...)
	response = strings.TrimSuffix(response, "\n")

	// If 'docker ps' failed due to some reason
	if err != nil {
		return fmt.Errorf("unable to confirm whether %s is running. error\n%v", containerName, err.Error())
	}
	// 'docker ps' worked fine, but the response did not have the container name
	if response == "" || response != containerName {
		return fmt.Errorf("container %s is not running", containerName)
	}

	return nil
}

func parseDockerError(component string, err error) error {
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode := exitError.ExitCode()
		if exitCode == 125 { //see https://github.com/moby/moby/pull/14012
			return fmt.Errorf("failed to launch %s. Is it already running?", component)
		}
		if exitCode == 127 {
			return fmt.Errorf("failed to launch %s. Make sure Docker is installed and running", component)
		}
	}
	return err
}

func isContainerRunError(err error) bool {
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode := exitError.ExitCode()
		return exitCode == 125
	}
	return false
}

func runPlacementService(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string, installLocation string) {
	defer wg.Done()

	image := fmt.Sprintf("%s:%s", daprDockerImageName, version)

	// Use only image for latest version
	if version == daprLatestVersion {
		image = daprDockerImageName
	}

	args := []string{
		"run",
		"--name", utils.CreateContainerName(DaprPlacementContainerName, dockerNetwork),
		"--restart", "always",
		"-d",
		"--entrypoint", "./placement",
	}

	if dockerNetwork != "" {
		args = append(args,
			"--network", dockerNetwork,
			"--network-alias", DaprPlacementContainerName)
	} else {
		osPort := 50005
		if runtime.GOOS == daprWindowsOS {
			osPort = 6050
		}

		args = append(args,
			"-p", fmt.Sprintf("%v:50005", osPort))
	}

	args = append(args, image)

	_, err := utils.RunCmdAndWait("docker", args...)

	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseDockerError("placement service", err)
		} else {
			errorChan <- fmt.Errorf("docker %s failed with: %v", args, err)
		}
		return
	}
	errorChan <- nil
}

func installDaprBinary(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string, installLocation string) {
	defer wg.Done()

	archiveExt := "tar.gz"
	if runtime.GOOS == daprWindowsOS {
		archiveExt = "zip"
	}

	if version == daprLatestVersion {
		var err error
		version, err = getLatestRelease(daprGitHubOrg, daprGitHubRepo)
		if err != nil {
			errorChan <- fmt.Errorf("cannot get the latest release version: %s", err)
			return
		}
		version = version[1:]
	}

	daprURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/v%s/%s_%s_%s.%s",
		daprGitHubOrg,
		daprGitHubRepo,
		version,
		daprRuntimeFilePrefix,
		runtime.GOOS,
		runtime.GOARCH,
		archiveExt)

	filepath, err := downloadFile(dir, daprURL)
	if err != nil {
		errorChan <- fmt.Errorf("error downloading Dapr binary: %s", err)
		return
	}

	extractedFilePath := ""

	if archiveExt == "zip" {
		extractedFilePath, err = unzip(filepath, dir)
	} else {
		extractedFilePath, err = untar(filepath, dir)
	}

	if err != nil {
		errorChan <- fmt.Errorf("error extracting Dapr binary: %s", err)
		return
	}

	daprPath, err := moveFileToPath(extractedFilePath, installLocation)
	if err != nil {
		errorChan <- fmt.Errorf("error moving Dapr binary to path: %s", err)
		return
	}

	err = makeExecutable(daprPath)
	if err != nil {
		errorChan <- fmt.Errorf("error making Dapr binary executable: %s", err)
		return
	}

	errorChan <- nil
}

func makeExecutable(filepath string) error {
	if runtime.GOOS != daprWindowsOS {
		err := os.Chmod(filepath, 0777)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzip(filepath, targetDir string) (string, error) {
	zipReader, err := zip.OpenReader(filepath)
	if err != nil {
		return "", err
	}

	if len(zipReader.Reader.File) > 0 {
		file := zipReader.Reader.File[0]

		zippedFile, err := file.Open()
		if err != nil {
			return "", err
		}
		defer zippedFile.Close()

		extractedFilePath := path.Join(
			targetDir,
			file.Name,
		)

		outputFile, err := os.OpenFile(
			extractedFilePath,
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			file.Mode(),
		)
		if err != nil {
			return "", err
		}
		defer outputFile.Close()

		_, err = io.Copy(outputFile, zippedFile)
		if err != nil {
			return "", err
		}

		return extractedFilePath, nil
	}

	return "", nil
}

func untar(filepath, targetDir string) (string, error) {
	tarFile, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer tarFile.Close()

	gzr, err := gzip.NewReader(tarFile)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return "", fmt.Errorf("file is empty")
		case err != nil:
			return "", err
		case header == nil:
			continue
		}

		extractedFilePath := path.Join(targetDir, header.Name)

		switch header.Typeflag {
		case tar.TypeReg:
			// Extract only daprd
			if header.Name != "daprd" {
				continue
			}

			f, err := os.OpenFile(extractedFilePath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return "", err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return "", err
			}
			f.Close()

			return extractedFilePath, nil
		}
	}
}

func moveFileToPath(filepath string, installLocation string) (string, error) {
	destDir := daprDefaultLinuxAndMacInstallPath
	if runtime.GOOS == daprWindowsOS {
		destDir = daprDefaultWindowsInstallPath
		filepath = strings.Replace(filepath, "/", "\\", -1)
	}

	fileName := path_filepath.Base(filepath)
	destFilePath := ""

	// if user specified --install-path, use that
	if installLocation != "" {
		destDir = installLocation
	}

	destFilePath = path.Join(destDir, fileName)

	input, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	fmt.Printf("Installing Dapr to %s\n", destDir)
	err = utils.CreateDirectory(destDir)
	if err != nil {
		return "", err
	}

	if err = ioutil.WriteFile(destFilePath, input, 0644); err != nil {
		if runtime.GOOS != daprWindowsOS && strings.Contains(err.Error(), "permission denied") {
			err = errors.New(err.Error() + " - please run with sudo")
		}
		return "", err
	}

	if runtime.GOOS == daprWindowsOS {
		p := os.Getenv("PATH")

		if !strings.Contains(strings.ToLower(p), strings.ToLower(destDir)) {
			_, err := utils.RunCmdAndWait("SETX", "PATH", p+fmt.Sprintf(";%s", destDir))
			if err != nil {
				return "", err
			}
		}

		return fmt.Sprintf("%s\\daprd.exe", destDir), nil
	}

	if installLocation != "" {
		color.Set(color.FgYellow)
		fmt.Printf("\nDapr installed to %s, please run the following to add it to your path:\n", destDir)
		fmt.Printf("    export PATH=$PATH:%s\n", destDir)
		color.Unset()
	}

	return destFilePath, nil
}

type githubRepoReleaseItem struct {
	URL     string `json:"url"`
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Draft   bool   `json:"draft"`
}

// nolint:gosec
func getLatestRelease(gitHubOrg, gitHubRepo string) (string, error) {
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", gitHubOrg, gitHubRepo)
	resp, err := http.Get(releaseURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("%s - %s", releaseURL, resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var githubRepoReleases []githubRepoReleaseItem
	err = json.Unmarshal(body, &githubRepoReleases)
	if err != nil {
		return "", err
	}

	if len(githubRepoReleases) == 0 {
		return "", fmt.Errorf("no releases")
	}

	for _, release := range githubRepoReleases {
		if !strings.Contains(release.TagName, "-rc") {
			return release.TagName, nil
		}
	}

	return "", fmt.Errorf("no releases")
}

// nolint:gosec
func downloadFile(dir string, url string) (string, error) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]

	filepath := path.Join(dir, fileName)
	_, err := os.Stat(filepath)
	if os.IsExist(err) {
		return "", nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", errors.New("runtime version not found")
	} else if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed with %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return filepath, nil
}
