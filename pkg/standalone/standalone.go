package standalone

import (
	"archive/zip"
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
)

const baseDownloadURL = "https://actionsreleases.blob.core.windows.net/bin"

// this should be configurable by versioning
const actionsImageURL = "yaron2/actionsedge:v2"

func Init() error {
	dir, err := getActionsDir()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errorChan := make(chan error)

	initSteps := []func(*sync.WaitGroup, chan<- error, string){}
	initSteps = append(initSteps, installActionsBinary)
	initSteps = append(initSteps, installAssignerBinary)
	initSteps = append(initSteps, installStateStore)

	wg.Add(len(initSteps))

	for _, step := range initSteps {
		go step(&wg, errorChan, dir)
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

func installStateStore(wg *sync.WaitGroup, errorChan chan<- error, dir string) {
	defer wg.Done()
	err := runCmd("docker", "run", "--restart", "always", "-d", "-p", "6379:6379", "redis")
	if err != nil {
		errorChan <- err
		return
	}
	errorChan <- nil
}

func installAssignerBinary(wg *sync.WaitGroup, errorChan chan<- error, dir string) {
	defer wg.Done()

	osPort := 50005
	if runtime.GOOS == "windows" {
		osPort = 6050
	}

	err := runCmd("docker", "run", "--restart", "always", "-d", "-p", fmt.Sprintf("%v:50005", osPort), "--entrypoint", "./assigner", actionsImageURL)
	if err != nil {
		errorChan <- err
		return
	}
	errorChan <- nil
}

func installActionsBinary(wg *sync.WaitGroup, errorChan chan<- error, dir string) {
	defer wg.Done()

	actionsURL := fmt.Sprintf("%s/action_%s_%s.zip", baseDownloadURL, runtime.GOOS, runtime.GOARCH)
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
	err := cmd.Start()
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
			runCmd("SETX", "PATH", p+";c:\\actions")
		}
		return path.Join(path_filepath.FromSlash("c:/actions"), fileName), nil
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
