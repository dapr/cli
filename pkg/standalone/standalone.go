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

	"github.com/briandowns/spinner"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

const (
	daprGitHubOrg              = "dapr"
	daprGitHubRepo             = "dapr"
	daprDockerImageName        = "daprio/dapr"
	daprRuntimeFilePrefix      = "daprd"
	daprWindowsOS              = "windows"
	daprLatestVersion          = "latest"
	DaprPlacementContainerName = "dapr_placement"
	DaprRedisContainerName     = "dapr_redis"
)

// Init installs Dapr on a local machine using the supplied runtimeVersion
func Init(runtimeVersion string, dockerNetwork string) error {
	dockerInstalled := isDockerInstalled()
	if !dockerInstalled {
		return errors.New("Could not connect to Docker.  Is Docker installed and running?")
	}

	dir, err := getDaprDir()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errorChan := make(chan error)

	initSteps := []func(*sync.WaitGroup, chan<- error, string, string, string){}
	initSteps = append(initSteps, installDaprBinary)
	initSteps = append(initSteps, runPlacementService)
	initSteps = append(initSteps, runRedis)

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
		go step(&wg, errorChan, dir, runtimeVersion, dockerNetwork)
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

func runRedis(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string) {
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
	err := utils.RunCmdAndWait("docker", args...)

	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseDockerError("Redis state store", err)
			return
		} else {
			errorChan <- fmt.Errorf("docker %s failed with: %v", args, err)
		}
	}
	errorChan <- nil
}

func parseDockerError(component string, err error) error {
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode := exitError.ExitCode()
		if exitCode == 125 { //see https://github.com/moby/moby/pull/14012
			return fmt.Errorf("Failed to launch %s. Is it already running?", component)
		}
		if exitCode == 127 {
			return fmt.Errorf("Failed to launch %s. Make sure Docker is installed and running", component)
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

func runPlacementService(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string) {
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

	err := utils.RunCmdAndWait("docker", args...)

	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseDockerError("placement service", err)
			return
		} else {
			errorChan <- fmt.Errorf("docker %s failed with: %v", args, err)
		}
	}
	errorChan <- nil
}

func installDaprBinary(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string) {
	defer wg.Done()

	archiveExt := "tar.gz"
	if runtime.GOOS == daprWindowsOS {
		archiveExt = "zip"
	}

	if version == daprLatestVersion {
		var err error
		version, err = getLatestRelease(daprGitHubOrg, daprGitHubRepo)
		if err != nil {
			errorChan <- fmt.Errorf("Cannot get the latest release version: %s", err)
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
		errorChan <- fmt.Errorf("Error downloading dapr binary: %s", err)
		return
	}

	extractedFilePath := ""
	err = nil

	if archiveExt == "zip" {
		extractedFilePath, err = unzip(filepath, dir)
	} else {
		extractedFilePath, err = untar(filepath, dir)
	}

	if err != nil {
		errorChan <- fmt.Errorf("Error extracting dapr binary: %s", err)
		return
	}

	daprPath, err := moveFileToPath(extractedFilePath)
	if err != nil {
		errorChan <- fmt.Errorf("Error moving dapr binary to path: %s", err)
		return
	}

	err = makeExecutable(daprPath)
	if err != nil {
		errorChan <- fmt.Errorf("Error making dapr binary executable: %s", err)
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

	for _, file := range zipReader.Reader.File {
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

func moveFileToPath(filepath string) (string, error) {
	fileName := path_filepath.Base(filepath)
	destFilePath := ""

	if runtime.GOOS == daprWindowsOS {
		p := os.Getenv("PATH")
		if !strings.Contains(strings.ToLower(string(p)), strings.ToLower("c:\\dapr")) {
			err := utils.RunCmdAndWait("SETX", "PATH", p+";c:\\dapr")
			if err != nil {
				return "", err
			}
		}
		return "c:\\dapr\\daprd.exe", nil
	}

	destFilePath = path.Join("/usr/local/bin", fileName)

	input, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(destFilePath, input, 0644)
	if err != nil {
		return "", err
	}

	return destFilePath, nil
}

type githubRepoReleaseItem struct {
	Url      string `json:"url"`
	Tag_name string `json:"tag_name"`
	Name     string `json:"name"`
	Draft    bool   `json:"draft"`
}

func getLatestRelease(gitHubOrg, gitHubRepo string) (string, error) {
	releaseUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", gitHubOrg, gitHubRepo)
	resp, err := http.Get(releaseUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("%s - %s", releaseUrl, resp.Status)
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
		return "", fmt.Errorf("No releases")
	}

	return githubRepoReleases[0].Tag_name, nil
}

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
