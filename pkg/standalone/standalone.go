package standalone

import (
	"archive/zip"
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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const baseDownloadURL = "https://actionsreleases.blob.core.windows.net/bin"
const actionsImageURL = "actionscore.azurecr.io/actions:merge"
const redisImageURL = "redis"

func Init() error {
	dockerClient, err := getDockerClient()
	if err != nil {
		return errors.New("Docker error. Make sure Docker is installed and running")
	}

	dir, err := getActionsDir()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errorChan := make(chan error)

	initSteps := []func(*sync.WaitGroup, chan<- error, string, *client.Client){}
	initSteps = append(initSteps, installActionsBinary)
	initSteps = append(initSteps, runPlacementService)
	initSteps = append(initSteps, runRedis)

	wg.Add(len(initSteps))

	for _, step := range initSteps {
		go step(&wg, errorChan, dir, dockerClient)
	}

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	for err := range errorChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func getActionsDir() (string, error) {
	p := ""

	if runtime.GOOS == "windows" {
		p = path_filepath.FromSlash("c:/actions")
	} else {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		p = path.Join(usr.HomeDir, ".actions")
	}

	err := os.MkdirAll(p, 0700)
	if err != nil {
		return "", err
	}

	return p, nil
}

func runRedis(wg *sync.WaitGroup, errorChan chan<- error, dir string, dockerClient *client.Client) {
	defer wg.Done()
	exists, err := containerExists("redis", dockerClient)
	if err != nil {
		errorChan <- fmt.Errorf("Docker error: %s", err)
		return
	}

	if !exists {
		err := runCmd("docker", "run", "--restart", "always", "-d", "-p", "6379:6379", redisImageURL)
		if err != nil {
			errorChan <- fmt.Errorf("Failed to launch Redis: %s", err)
			return
		}
	}

	errorChan <- nil
}

func getDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.39"))
}

func getContainersList(client *client.Client) ([]types.Container, error) {
	return client.ContainerList(context.Background(), types.ContainerListOptions{})
}

func containerExists(image string, client *client.Client) (bool, error) {
	containers, err := getContainersList(client)
	if err != nil {
		return false, err
	}

	for _, c := range containers {
		if c.Image == actionsImageURL {
			return true, nil
		}
	}
	return false, nil
}

func runPlacementService(wg *sync.WaitGroup, errorChan chan<- error, dir string, dockerClient *client.Client) {
	defer wg.Done()
	exists, err := containerExists(actionsImageURL, dockerClient)
	if err != nil {
		errorChan <- fmt.Errorf("Docker error: %s", err)
		return
	}

	if !exists {
		osPort := 50005
		if runtime.GOOS == "windows" {
			osPort = 6050
		}

		err := runCmd("docker", "run", "--restart", "always", "-d", "-p", fmt.Sprintf("%v:50005", osPort), "--entrypoint", "./placement", actionsImageURL)
		if err != nil {
			errorChan <- fmt.Errorf("Failed to launch placement service: %s", err)
			return
		}
	}
	errorChan <- nil
}

func installActionsBinary(wg *sync.WaitGroup, errorChan chan<- error, dir string, dockerClient *client.Client) {
	defer wg.Done()

	actionsURL := fmt.Sprintf("%s/actionsrt_%s_%s.zip", baseDownloadURL, runtime.GOOS, runtime.GOARCH)
	filepath, err := downloadFile(dir, actionsURL)
	if err != nil {
		errorChan <- fmt.Errorf("Error downloading actions binary: %s", err)
		return
	}

	extractedFilePath, err := extractFile(filepath, dir)
	if err != nil {
		errorChan <- fmt.Errorf("Error extracting actions binary: %s", err)
		return
	}

	actionsPath, err := moveFileToPath(extractedFilePath)
	if err != nil {
		errorChan <- fmt.Errorf("Error moving actions binary to path: %s", err)
		return
	}

	err = makeExecutable(actionsPath)
	if err != nil {
		errorChan <- fmt.Errorf("Error making actions binary executable: %s", err)
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

func extractFile(filepath, targetDir string) (string, error) {
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

func moveFileToPath(filepath string) (string, error) {
	fileName := path_filepath.Base(filepath)
	destFilePath := ""

	if runtime.GOOS == "windows" {
		p := os.Getenv("PATH")
		if !strings.Contains(strings.ToLower(string(p)), strings.ToLower("c:\\actions")) {
			err := runCmd("SETX", "PATH", p+";c:\\actions")
			if err != nil {
				return "", err
			}
		}
		return "c:\\actions\\actionsrt.exe", nil
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
