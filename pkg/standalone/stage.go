package standalone

import (
	"errors"
	"fmt"
	"os"
	path_filepath "path/filepath"
	"strings"
	"sync"

	"github.com/dapr/cli/pkg/print"
	cli_ver "github.com/dapr/cli/pkg/version"
	"github.com/dapr/cli/utils"
)

const (
	stagingBaseDir   = "pkg/standalone/staging"
	runtimeVerFile   = "runtime.ver"
	dashboardVerFile = "dashboard.ver"
)

var stagingBinDir = path_filepath.Join(stagingBaseDir, defaultDaprBinDirName)

func Stage(rv, dv string) error {
	dockerInstalled := utils.IsDockerInstalled()
	if !dockerInstalled {
		return errors.New("could not connect to Docker. Docker may not be installed or running")
	}

	var err error

	runtimeVersion, err = findRuntimeVersion(rv)
	if err != nil {
		return err
	}
	print.InfoStatusEvent(os.Stdout, "Staging runtime version %s", runtimeVersion)
	dashboardVersion = findDashboardVersion(dv)

	err = prepareDaprInstallDir(stagingBinDir)
	if err != nil {
		return err
	}

	err = writeVersions(runtimeVersion, dashboardVersion)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errorChan := make(chan error)
	initSteps := []func(*sync.WaitGroup, chan<- error){
		saveDaprImage,
		saveRedisImage,
		saveZipkinImage,
		downloadDaprd,
		downloadDashboard,
		downloadPlacement,
	}

	wg.Add(len(initSteps))

	msg := "Downloading binaries and images..."
	stopSpinning := print.Spinner(os.Stdout, msg)
	defer stopSpinning(print.Failure)

	for _, step := range initSteps {
		// Run init on the configurations and containers
		go step(&wg, errorChan)
	}

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	for err = range errorChan {
		if err != nil {
			return err
		}
	}

	stopSpinning(print.Success)

	msg = "Dapr binaries and images are downloaded."
	print.SuccessStatusEvent(os.Stdout, msg)
	print.InfoStatusEvent(os.Stdout, "%s binary has been installed to %s.", daprRuntimeFilePrefix, stagingBinDir)
	return nil
}

func downloadDaprd(wg *sync.WaitGroup, errorChan chan<- error) {
	defer wg.Done()
	_, err := downloadBinary(stagingBinDir, runtimeVersion, daprRuntimeFilePrefix, cli_ver.DaprGitHubRepo)
	errorChan <- err
}

func downloadDashboard(wg *sync.WaitGroup, errorChan chan<- error) {
	defer wg.Done()
	_, err := downloadBinary(stagingBinDir, dashboardVersion, dashboardFilePrefix, cli_ver.DashboardGitHubRepo)
	errorChan <- err
}

func downloadPlacement(wg *sync.WaitGroup, errorChan chan<- error) {
	defer wg.Done()
	_, err := downloadBinary(stagingBinDir, runtimeVersion, placementServiceFilePrefix, cli_ver.DaprGitHubRepo)
	errorChan <- err
}

func saveDaprImage(wg *sync.WaitGroup, errorChan chan<- error) {
	defer wg.Done()
	image := fmt.Sprintf("%s:%s", daprDockerImageName, runtimeVersion)
	// Use only image for latest version
	if runtimeVersion == latestVersion {
		image = daprDockerImageName
	}

	errorChan <- saveImage(image)
}

func saveZipkinImage(wg *sync.WaitGroup, errorChan chan<- error) {
	defer wg.Done()
	errorChan <- saveImage(zipkinDockerImageName)
}

func saveRedisImage(wg *sync.WaitGroup, errorChan chan<- error) {
	defer wg.Done()
	errorChan <- saveImage(redisDockerImageName)
}

func saveImage(image string) error {
	_, err := utils.RunCmdAndWait("docker", "pull", image)
	if err != nil {
		return err
	}

	imageDir := path_filepath.Join("pkg/standalone/staging", "images")
	err = prepareDaprInstallDir(imageDir)
	if err != nil {
		return err
	}

	filename := imageFileName(image)
	_, err = utils.RunCmdAndWait("docker", "save", "-o", path_filepath.Join(imageDir, filename), image)
	return err
}

func imageFileName(image string) string {
	filename := image + ".tar.gz"
	filename = strings.ReplaceAll(filename, "/", "-")
	filename = strings.ReplaceAll(filename, ":", "-")
	return filename
}

func writeVersions(runtimeVersion, dashboardVersion string) error {
	err := writeFile(path_filepath.Join(stagingBaseDir, runtimeVerFile), runtimeVersion)
	if err != nil {
		return err
	}

	err = writeFile(path_filepath.Join(stagingBaseDir, dashboardVerFile), dashboardVersion)
	if err != nil {
		return err
	}

	return nil
}

func writeFile(filePath string, content string) error {
	f, err := os.Create(filePath)
	defer func() {
		_ = f.Close()
	}()

	if err != nil {
		print.WarningStatusEvent(os.Stdout, "cannot create %s: %v", filePath, err)
		return err
	}

	_, err = f.WriteString(content)
	if err != nil {
		print.WarningStatusEvent(os.Stdout, "cannot write %s: %v", filePath, err)
		return err
	}

	return nil
}
