package standalone

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
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
)

const baseDownloadURL = "https://daprreleases.blob.core.windows.net/release"
const daprImageURL = "actionscore.azurecr.io/dapr"

func Init(runtimeVersion string) error {
	dockerInstalled := isDockerInstalled()
	if !dockerInstalled {
		return errors.New("Could not connect to Docker.  Is Docker is installed and running?")
	}

	dir, err := getDaprDir()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errorChan := make(chan error)

	initSteps := []func(*sync.WaitGroup, chan<- error, string, string){}
	initSteps = append(initSteps, installDaprBinary)
	initSteps = append(initSteps, runPlacementService)
	initSteps = append(initSteps, runRedis)

	wg.Add(len(initSteps))

	msg := "Downloading binaries and setting up components..."
	var s *spinner.Spinner
	if runtime.GOOS == "windows" {
		print.InfoStatusEvent(os.Stdout, msg)
	} else {
		s = spinner.New(spinner.CharSets[0], 100*time.Millisecond)
		s.Writer = os.Stdout
		s.Color("cyan")
		s.Suffix = fmt.Sprintf("  %s", msg)
		s.Start()
	}

	for _, step := range initSteps {
		go step(&wg, errorChan, dir, runtimeVersion)
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

	if runtime.GOOS == "windows" {
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

func runRedis(wg *sync.WaitGroup, errorChan chan<- error, dir, version string) {
	defer wg.Done()
	err := runCmd("docker", "run", "--restart", "always", "-d", "-p", "6379:6379", "redis")
	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseDockerError("Redis state store", err)
			return
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

func runPlacementService(wg *sync.WaitGroup, errorChan chan<- error, dir, version string) {
	defer wg.Done()

	osPort := 50005
	if runtime.GOOS == "windows" {
		osPort = 6050
	}

	image := fmt.Sprintf("%s:%s", daprImageURL, version)
	err := runCmd("docker", "run", "--restart", "always", "-d", "-p", fmt.Sprintf("%v:50005", osPort), "--entrypoint", "./placement", image)
	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseDockerError("placement service", err)
			return
		}
	}
	errorChan <- nil
}

func installDaprBinary(wg *sync.WaitGroup, errorChan chan<- error, dir, version string) {
	defer wg.Done()

	archiveExt := "tar.gz"
	if runtime.GOOS == "windows" || strings.Contains(version, "0.3.0-alpha") /* only 0.3.0-alpha uses zip for all OSs */ {
		archiveExt = "zip"
	}

	daprURL := fmt.Sprintf("%s/%s/daprd_%s_%s.%s", baseDownloadURL, version, runtime.GOOS, runtime.GOARCH, archiveExt)
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
	if runtime.GOOS != "windows" {
		err := os.Chmod(filepath, 0777)
		if err != nil {
			return err
		}
	}

	return nil
}

func runCmd(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	err := cmd.Run()
	if err != nil {
		return err
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

	if runtime.GOOS == "windows" {
		p := os.Getenv("PATH")
		if !strings.Contains(strings.ToLower(string(p)), strings.ToLower("c:\\dapr")) {
			err := runCmd("SETX", "PATH", p+";c:\\dapr")
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
