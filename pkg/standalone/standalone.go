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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	path_filepath "path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"

	"github.com/dapr/cli/pkg/print"
	cli_ver "github.com/dapr/cli/pkg/version"
	"github.com/dapr/cli/utils"
)

const (
	daprRuntimeFilePrefix      = "daprd"
	dashboardFilePrefix        = "dashboard"
	placementServiceFilePrefix = "placement"

	daprWindowsOS = "windows"

	latestVersion   = "latest"
	daprDefaultHost = "localhost"

	pubSubYamlFileName     = "pubsub.yaml"
	stateStoreYamlFileName = "statestore.yaml"

	// used when DAPR_DEFAULT_IMAGE_REGISTRY is not set.
	dockerContainerRegistryName = "dockerhub"
	daprDockerImageName         = "daprio/dapr"
	redisDockerImageName        = "redis"
	zipkinDockerImageName       = "openzipkin/zipkin"

	// used when DAPR_DEFAULT_IMAGE_REGISTRY is set as GHCR.
	githubContainerRegistryName = "ghcr"
	ghcrURI                     = "ghcr.io/dapr"
	daprGhcrImageName           = "dapr"
	redisGhcrImageName          = "3rdparty/redis"
	zipkinGhcrImageName         = "3rdparty/zipkin"

	// DaprPlacementContainerName is the container name of placement service.
	DaprPlacementContainerName = "dapr_placement"
	// DaprRedisContainerName is the container name of redis.
	DaprRedisContainerName = "dapr_redis"
	// DaprZipkinContainerName is the container name of zipkin.
	DaprZipkinContainerName = "dapr_zipkin"

	errInstallTemplate = "please run `dapr uninstall` first before running `dapr init`"
)

var (
	defaultImageRegistryName string
	privateRegTemplateString = "%s/dapr/%s"
	isAirGapInit             bool
)

type configuration struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Tracing struct {
			SamplingRate string `yaml:"samplingRate,omitempty"`
			Zipkin       struct {
				EndpointAddress string `yaml:"endpointAddress,omitempty"`
			} `yaml:"zipkin,omitempty"`
		} `yaml:"tracing,omitempty"`
	} `yaml:"spec"`
}

type component struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Type     string                  `yaml:"type"`
		Version  string                  `yaml:"version"`
		Metadata []componentMetadataItem `yaml:"metadata"`
	} `yaml:"spec"`
}

type componentMetadataItem struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type initInfo struct {
	fromDir          string
	bundleDet        *bundleDetails
	slimMode         bool
	runtimeVersion   string
	dashboardVersion string
	dockerNetwork    string
	imageRegistryURL string
}

type daprImageInfo struct {
	ghcrImageName      string
	dockerHubImageName string
	imageRegistryURL   string
	imageRegistryName  string
}

// Check if the previous version is already installed.
func isBinaryInstallationRequired(binaryFilePrefix, installDir string) (bool, error) {
	binaryPath := binaryFilePath(installDir, binaryFilePrefix)

	// first time install?
	_, err := os.Stat(binaryPath)
	if !os.IsNotExist(err) {
		return false, fmt.Errorf("%s %w, %s", binaryPath, os.ErrExist, errInstallTemplate)
	}
	return true, nil
}

// Init installs Dapr on a local machine using the supplied runtimeVersion.
func Init(runtimeVersion, dashboardVersion string, dockerNetwork string, slimMode bool, imageRegistryURL string, fromDir string) error {
	var err error
	var bundleDet bundleDetails
	fromDir = strings.TrimSpace(fromDir)
	// AirGap init flow is true when fromDir var is set i.e. --from-dir flag has value.
	setAirGapInit(fromDir)
	if !slimMode {
		// If --slim installation is not requested, check if docker is installed.
		dockerInstalled := utils.IsDockerInstalled()
		if !dockerInstalled {
			return errors.New("could not connect to Docker. Docker may not be installed or running")
		}

		// Initialize default registry only if any of --slim or --image-registry or --from-dir are not given.
		if len(strings.TrimSpace(imageRegistryURL)) == 0 && !isAirGapInit {
			defaultImageRegistryName, err = utils.GetDefaultRegistry(githubContainerRegistryName, dockerContainerRegistryName)
			if err != nil {
				return err
			}
		}
	}

	// Set runtime version.

	if runtimeVersion == latestVersion && !isAirGapInit {
		runtimeVersion, err = cli_ver.GetDaprVersion()
		if err != nil {
			return fmt.Errorf("cannot get the latest release version: '%w'. Try specifying --runtime-version=<desired_version>", err)
		}
	}

	if dashboardVersion == latestVersion && !isAirGapInit {
		dashboardVersion, err = cli_ver.GetDashboardVersion()
		if err != nil {
			print.WarningStatusEvent(os.Stdout, "cannot get the latest dashboard version: '%s'. Try specifying --dashboard-version=<desired_version>", err)
			print.WarningStatusEvent(os.Stdout, "continuing, but dashboard will be unavailable")
		}
	}

	// If --from-dir flag is given try parsing the details from the expected details file in the specified directory.
	if isAirGapInit {
		bundleDet = bundleDetails{}
		detailsFilePath := path_filepath.Join(fromDir, bundleDetailsFileName)
		err = bundleDet.readAndParseDetails(detailsFilePath)
		if err != nil {
			return fmt.Errorf("error parsing details file from bundle location: %w", err)
		}

		// Set runtime and dashboard versions from the bundle details parsed.

		runtimeVersion = *bundleDet.RuntimeVersion
		dashboardVersion = *bundleDet.DashboardVersion
	}

	// At this point the runtimeVersion variable is parsed either from the details file if --fromDir is specified or
	// got from running the command cli_ver.GetRuntimeVersion().

	// After this point runtimeVersion will not be latest string but rather actual version.

	print.InfoStatusEvent(os.Stdout, "Installing runtime version %s", runtimeVersion)

	daprBinDir := defaultDaprBinPath()
	err = prepareDaprInstallDir(daprBinDir)
	if err != nil {
		return err
	}

	// confirm if installation is required.
	if ok, er := isBinaryInstallationRequired(daprRuntimeFilePrefix, daprBinDir); !ok {
		return er
	}

	var wg sync.WaitGroup
	errorChan := make(chan error)
	initSteps := []func(*sync.WaitGroup, chan<- error, initInfo){
		createSlimConfiguration,
		createComponentsAndConfiguration,
		installDaprRuntime,
		installPlacement,
		installDashboard,
		runPlacementService,
		runRedis,
		runZipkin,
	}

	// Init other configurations, containers.
	wg.Add(len(initSteps))

	msg := "Downloading binaries and setting up components..."
	if isAirGapInit {
		msg = "Extracting binaries and setting up components..."
	}
	stopSpinning := print.Spinner(os.Stdout, msg)
	defer stopSpinning(print.Failure)

	// Make default components directory.
	err = makeDefaultComponentsDir()
	if err != nil {
		return err
	}

	info := initInfo{
		// values in bundleDet can be nil if fromDir is empty, so must be used in conjunction with fromDir.
		bundleDet:        &bundleDet,
		fromDir:          fromDir,
		slimMode:         slimMode,
		runtimeVersion:   runtimeVersion,
		dashboardVersion: dashboardVersion,
		dockerNetwork:    dockerNetwork,
		imageRegistryURL: imageRegistryURL,
	}
	for _, step := range initSteps {
		// Run init on the configurations and containers.
		go step(&wg, errorChan, info)
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

	stopSpinning(print.Success)

	msg = "Downloaded binaries and completed components set up."
	if isAirGapInit {
		msg = "Extracted binaries and completed components set up."
	}
	print.SuccessStatusEvent(os.Stdout, msg)
	print.InfoStatusEvent(os.Stdout, "%s binary has been installed to %s.", daprRuntimeFilePrefix, daprBinDir)
	if slimMode {
		// Print info on placement binary only on slim install.
		print.InfoStatusEvent(os.Stdout, "%s binary has been installed to %s.", placementServiceFilePrefix, daprBinDir)
	} else {
		dockerContainerNames := []string{DaprPlacementContainerName, DaprRedisContainerName, DaprZipkinContainerName}
		// Skip redis and zipkin in local installation mode.
		if isAirGapInit {
			dockerContainerNames = []string{DaprPlacementContainerName}
		}
		for _, container := range dockerContainerNames {
			containerName := utils.CreateContainerName(container, dockerNetwork)
			ok, err := confirmContainerIsRunningOrExists(containerName, true)
			if err != nil {
				return err
			}
			if ok {
				print.InfoStatusEvent(os.Stdout, "%s container is running.", containerName)
			}
		}
		print.InfoStatusEvent(os.Stdout, "Use `docker ps` to check running containers.")
	}
	return nil
}

func runZipkin(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if info.slimMode || isAirGapInit {
		return
	}

	zipkinContainerName := utils.CreateContainerName(DaprZipkinContainerName, info.dockerNetwork)

	exists, err := confirmContainerIsRunningOrExists(zipkinContainerName, false)
	if err != nil {
		errorChan <- err
		return
	}
	args := []string{}

	var imageName string

	if exists {
		// do not create container again if it exists.
		args = append(args, "start", zipkinContainerName)
	} else {
		imageName, err = resolveImageURI(daprImageInfo{
			ghcrImageName:      zipkinGhcrImageName,
			dockerHubImageName: zipkinDockerImageName,
			imageRegistryURL:   info.imageRegistryURL,
			imageRegistryName:  defaultImageRegistryName,
		})
		if err != nil {
			errorChan <- err
			return
		}

		args = append(args,
			"run",
			"--name", zipkinContainerName,
			"--restart", "always",
			"-d",
		)

		if info.dockerNetwork != "" {
			args = append(
				args,
				"--network", info.dockerNetwork,
				"--network-alias", DaprZipkinContainerName)
		} else {
			args = append(
				args,
				"-p", "9411:9411")
		}

		args = append(args, imageName)
	}
	_, err = utils.RunCmdAndWait("docker", args...)

	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseDockerError("Zipkin tracing", err)
		} else {
			errorChan <- fmt.Errorf("docker %s failed with: %w", args, err)
		}
		return
	}
	errorChan <- nil
}

func runRedis(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if info.slimMode || isAirGapInit {
		return
	}

	redisContainerName := utils.CreateContainerName(DaprRedisContainerName, info.dockerNetwork)

	exists, err := confirmContainerIsRunningOrExists(redisContainerName, false)
	if err != nil {
		errorChan <- err
		return
	}
	args := []string{}

	var imageName string
	if exists {
		// do not create container again if it exists.
		args = append(args, "start", redisContainerName)
	} else {
		imageName, err = resolveImageURI(daprImageInfo{
			ghcrImageName:      redisGhcrImageName,
			dockerHubImageName: redisDockerImageName,
			imageRegistryURL:   info.imageRegistryURL,
			imageRegistryName:  defaultImageRegistryName,
		})
		if err != nil {
			errorChan <- err
			return
		}

		args = append(args,
			"run",
			"--name", redisContainerName,
			"--restart", "always",
			"-d",
		)

		if info.dockerNetwork != "" {
			args = append(
				args,
				"--network", info.dockerNetwork,
				"--network-alias", DaprRedisContainerName)
		} else {
			args = append(
				args,
				"-p", "6379:6379")
		}
		args = append(args, imageName)
	}
	_, err = utils.RunCmdAndWait("docker", args...)

	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseDockerError("Redis state store", err)
		} else {
			errorChan <- fmt.Errorf("docker %s failed with: %w", args, err)
		}
		return
	}
	errorChan <- nil
}

func runPlacementService(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if info.slimMode {
		return
	}

	placementContainerName := utils.CreateContainerName(DaprPlacementContainerName, info.dockerNetwork)

	exists, err := confirmContainerIsRunningOrExists(placementContainerName, false)

	if err != nil {
		errorChan <- err
		return
	} else if exists {
		errorChan <- fmt.Errorf("%s container exists or is running. %s", placementContainerName, errInstallTemplate)
		return
	}
	var image string

	imgInfo := daprImageInfo{
		ghcrImageName:      daprGhcrImageName,
		dockerHubImageName: daprDockerImageName,
		imageRegistryURL:   info.imageRegistryURL,
		imageRegistryName:  defaultImageRegistryName,
	}

	if isAirGapInit {
		// if --from-dir flag is given load the image details from the installer-bundle.
		dir := path_filepath.Join(info.fromDir, *info.bundleDet.ImageSubDir)
		image = info.bundleDet.getPlacementImageName()
		err = loadDocker(dir, info.bundleDet.getPlacementImageFileName())
		if err != nil {
			errorChan <- err
			return
		}
	} else {
		// otherwise load the image from the specified repository.
		image, err = getPlacementImageName(imgInfo, info)
		if err != nil {
			errorChan <- err
			return
		}
	}

	args := []string{
		"run",
		"--name", placementContainerName,
		"--restart", "always",
		"-d",
		"--entrypoint", "./placement",
	}

	if info.dockerNetwork != "" {
		args = append(args,
			"--network", info.dockerNetwork,
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

	_, err = utils.RunCmdAndWait("docker", args...)

	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseDockerError("placement service", err)
		} else {
			errorChan <- fmt.Errorf("docker %s failed with: %w", args, err)
		}
		return
	}
	errorChan <- nil
}

func moveDashboardFiles(extractedFilePath string, dir string) (string, error) {
	// Move /release/os/web directory to /web.
	oldPath := path_filepath.Join(path_filepath.Dir(extractedFilePath), "web")
	newPath := path_filepath.Join(dir, "web")
	err := os.Rename(oldPath, newPath)
	if err != nil {
		err = fmt.Errorf("failed to move dashboard files: %w", err)
		return "", err
	}

	// Move binary from /release/<os>/web/dashboard(.exe) to /dashboard(.exe).
	err = os.Rename(extractedFilePath, path_filepath.Join(dir, path_filepath.Base(extractedFilePath)))
	if err != nil {
		err = fmt.Errorf("error moving %s binary to path: %w", path_filepath.Base(extractedFilePath), err)
		return "", err
	}

	// Change the extracted binary file path to reflect the move above.
	extractedFilePath = path_filepath.Join(dir, path_filepath.Base(extractedFilePath))

	// Remove the now-empty 'release' directory.
	err = os.RemoveAll(path_filepath.Join(dir, "release"))
	if err != nil {
		err = fmt.Errorf("error moving dashboard files: %w", err)
		return "", err
	}

	return extractedFilePath, nil
}

func installDaprRuntime(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	err := installBinary(info.runtimeVersion, daprRuntimeFilePrefix, cli_ver.DaprGitHubRepo, info)
	if err != nil {
		errorChan <- err
	}
}

func installDashboard(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()
	if info.dashboardVersion == "" {
		return
	}

	err := installBinary(info.dashboardVersion, dashboardFilePrefix, cli_ver.DashboardGitHubRepo, info)
	if err != nil {
		errorChan <- err
	}
}

func installPlacement(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if !info.slimMode {
		return
	}

	err := installBinary(info.runtimeVersion, placementServiceFilePrefix, cli_ver.DaprGitHubRepo, info)
	if err != nil {
		errorChan <- err
	}
}

// installBinary installs the daprd, placement or dashboard binaries and associated files inside the default dapr bin directory.
func installBinary(version, binaryFilePrefix, githubRepo string, info initInfo) error {
	var (
		err      error
		filepath string
	)

	dir := defaultDaprBinPath()
	if isAirGapInit {
		filepath = path_filepath.Join(info.fromDir, *info.bundleDet.BinarySubDir, binaryName(binaryFilePrefix))
	} else {
		filepath, err = downloadBinary(dir, version, binaryFilePrefix, githubRepo)
		if err != nil {
			return fmt.Errorf("error downloading %s binary: %w", binaryFilePrefix, err)
		}
	}

	extractedFilePath, err := extractFile(filepath, dir, binaryFilePrefix)
	if err != nil {
		return err
	}

	// remove downloaded archive from the default dapr bin path.
	if !isAirGapInit {
		err = os.Remove(filepath)
		if err != nil {
			return fmt.Errorf("failed to remove archive: %w", err)
		}
	}

	if binaryFilePrefix == "dashboard" {
		extractedFilePath, err = moveDashboardFiles(extractedFilePath, dir)
		if err != nil {
			return err
		}
	}

	binaryPath, err := moveFileToPath(extractedFilePath, dir)
	if err != nil {
		return fmt.Errorf("error moving %s binary to path: %w", binaryFilePrefix, err)
	}

	err = makeExecutable(binaryPath)
	if err != nil {
		return fmt.Errorf("error making %s binary executable: %w", binaryFilePrefix, err)
	}

	return nil
}

func createComponentsAndConfiguration(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if info.slimMode || isAirGapInit {
		return
	}

	redisHost := daprDefaultHost
	zipkinHost := daprDefaultHost
	if info.dockerNetwork != "" {
		// Default to network scoped alias of the container names when a dockerNetwork is specified.
		redisHost = DaprRedisContainerName
		zipkinHost = DaprZipkinContainerName
	}
	var err error

	// Make default components directory.
	componentsDir := DefaultComponentsDirPath()

	err = createRedisPubSub(redisHost, componentsDir)
	if err != nil {
		errorChan <- fmt.Errorf("error creating redis pubsub component file: %w", err)
		return
	}
	err = createRedisStateStore(redisHost, componentsDir)
	if err != nil {
		errorChan <- fmt.Errorf("error creating redis statestore component file: %w", err)
		return
	}
	err = createDefaultConfiguration(zipkinHost, DefaultConfigFilePath())
	if err != nil {
		errorChan <- fmt.Errorf("error creating default configuration file: %w", err)
		return
	}
}

func createSlimConfiguration(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if !(info.slimMode || isAirGapInit) {
		return
	}

	// For --slim we pass empty string so that we do not configure zipkin.
	err := createDefaultConfiguration("", DefaultConfigFilePath())
	if err != nil {
		errorChan <- fmt.Errorf("error creating default configuration file: %w", err)
		return
	}
}

func makeDefaultComponentsDir() error {
	// Make default components directory.
	componentsDir := DefaultComponentsDirPath()
	//nolint
	_, err := os.Stat(componentsDir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(componentsDir, 0o755)
		if errDir != nil {
			return fmt.Errorf("error creating default components folder: %w", errDir)
		}
	}

	os.Chmod(componentsDir, 0o777)
	return nil
}

func makeExecutable(filepath string) error {
	if runtime.GOOS != daprWindowsOS {
		err := os.Chmod(filepath, 0o777)
		if err != nil {
			return err
		}
	}

	return nil
}

// https://github.com/snyk/zip-slip-vulnerability, fixes gosec G305
func sanitizeExtractPath(destination string, filePath string) (string, error) {
	destpath := path_filepath.Join(destination, filePath)
	if !strings.HasPrefix(destpath, path_filepath.Clean(destination)+string(os.PathSeparator)) {
		return "", fmt.Errorf("%s: illegal file path", filePath)
	}
	return destpath, nil
}

func extractFile(filepath, dir, binaryFilePrefix string) (string, error) {
	var extractFunc func(string, string, string) (string, error)
	if archiveExt() == "zip" {
		extractFunc = unzipExternalFile
	} else {
		extractFunc = untarExternalFile
	}

	extractedFilePath, err := extractFunc(filepath, dir, binaryFilePrefix)
	if err != nil {
		return "", fmt.Errorf("error extracting %s binary: %w", binaryFilePrefix, err)
	}

	return extractedFilePath, nil
}

func unzipExternalFile(filepath, dir, binaryFilePrefix string) (string, error) {
	r, err := zip.OpenReader(filepath)
	if err != nil {
		return "", fmt.Errorf("error open zip file %s: %w", filepath, err)
	}
	defer r.Close()

	return unzip(&r.Reader, dir, binaryFilePrefix)
}

func unzip(r *zip.Reader, targetDir string, binaryFilePrefix string) (string, error) {
	foundBinary := ""
	for _, f := range r.File {
		fpath, err := sanitizeExtractPath(targetDir, f.Name)
		if err != nil {
			return "", err
		}

		if strings.HasSuffix(fpath, fmt.Sprintf("%s.exe", binaryFilePrefix)) {
			foundBinary = fpath
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(path_filepath.Dir(fpath), os.ModePerm); err != nil {
			return "", err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", err
		}

		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		// #nosec G110
		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return "", err
		}
	}
	return foundBinary, nil
}

func untarExternalFile(filepath, dir, binaryFilePrefix string) (string, error) {
	reader, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("error open tar gz file %s: %w", filepath, err)
	}
	defer reader.Close()

	return untar(reader, dir, binaryFilePrefix)
}

func untar(reader io.Reader, targetDir string, binaryFilePrefix string) (string, error) {
	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	foundBinary := ""
	for {
		header, err := tr.Next()
		//nolint
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		} else if header == nil {
			continue
		}

		// untar all files in archive.
		path, err := sanitizeExtractPath(targetDir, header.Name)
		if err != nil {
			return "", err
		}

		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return "", err
			}
			continue
		}

		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
		if err != nil {
			return "", err
		}
		defer f.Close()

		// #nosec G110
		if _, err = io.Copy(f, tr); err != nil {
			return "", err
		}

		// If the found file is the binary that we want to find, save it and return later.
		if strings.HasSuffix(header.Name, binaryFilePrefix) {
			foundBinary = path
		}
	}
	return foundBinary, nil
}

func moveFileToPath(filepath string, installLocation string) (string, error) {
	fileName := path_filepath.Base(filepath)
	destFilePath := ""

	destDir := installLocation
	destFilePath = path.Join(destDir, fileName)

	input, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	err = utils.CreateDirectory(destDir)
	if err != nil {
		return "", err
	}

	// #nosec G306
	if err = ioutil.WriteFile(destFilePath, input, 0o644); err != nil {
		if runtime.GOOS != daprWindowsOS && strings.Contains(err.Error(), "permission denied") {
			err = errors.New(err.Error() + " - please run with sudo")
		}
		return "", err
	}

	if runtime.GOOS == daprWindowsOS {
		p := os.Getenv("PATH")

		if !strings.Contains(strings.ToLower(p), strings.ToLower(destDir)) {
			pathCmd := "[System.Environment]::SetEnvironmentVariable('Path',[System.Environment]::GetEnvironmentVariable('Path','user') + '" + fmt.Sprintf(";%s", destDir) + "', 'user')"
			_, err := utils.RunCmdAndWait("powershell", pathCmd)
			if err != nil {
				return "", err
			}
		}

		return fmt.Sprintf("%s\\daprd.exe", destDir), nil
	}

	if strings.HasPrefix(fileName, daprRuntimeFilePrefix) && installLocation != "" {
		color.Set(color.FgYellow)
		fmt.Printf("\nDapr runtime installed to %s, you may run the following to add it to your path if you want to run daprd directly:\n", destDir)
		fmt.Printf("    export PATH=$PATH:%s\n", destDir)
		color.Unset()
	}

	return destFilePath, nil
}

func createRedisStateStore(redisHost string, componentsPath string) error {
	redisStore := component{
		APIVersion: "dapr.io/v1alpha1",
		Kind:       "Component",
	}

	redisStore.Metadata.Name = "statestore"
	redisStore.Spec.Type = "state.redis"
	redisStore.Spec.Version = "v1"
	redisStore.Spec.Metadata = []componentMetadataItem{
		{
			Name:  "redisHost",
			Value: fmt.Sprintf("%s:6379", redisHost),
		},
		{
			Name:  "redisPassword",
			Value: "",
		},
		{
			Name:  "actorStateStore",
			Value: "true",
		},
	}

	b, err := yaml.Marshal(&redisStore)
	if err != nil {
		return err
	}

	filePath := path_filepath.Join(componentsPath, stateStoreYamlFileName)
	err = checkAndOverWriteFile(filePath, b)

	return err
}

func createRedisPubSub(redisHost string, componentsPath string) error {
	redisPubSub := component{
		APIVersion: "dapr.io/v1alpha1",
		Kind:       "Component",
	}

	redisPubSub.Metadata.Name = "pubsub"
	redisPubSub.Spec.Type = "pubsub.redis"
	redisPubSub.Spec.Version = "v1"
	redisPubSub.Spec.Metadata = []componentMetadataItem{
		{
			Name:  "redisHost",
			Value: fmt.Sprintf("%s:6379", redisHost),
		},
		{
			Name:  "redisPassword",
			Value: "",
		},
	}

	b, err := yaml.Marshal(&redisPubSub)
	if err != nil {
		return err
	}

	filePath := path_filepath.Join(componentsPath, pubSubYamlFileName)
	err = checkAndOverWriteFile(filePath, b)

	return err
}

func createDefaultConfiguration(zipkinHost, filePath string) error {
	defaultConfig := configuration{
		APIVersion: "dapr.io/v1alpha1",
		Kind:       "Configuration",
	}
	defaultConfig.Metadata.Name = "daprConfig"
	if zipkinHost != "" {
		defaultConfig.Spec.Tracing.SamplingRate = "1"
		defaultConfig.Spec.Tracing.Zipkin.EndpointAddress = fmt.Sprintf("http://%s:9411/api/v2/spans", zipkinHost) //nolint:nosprintfhostport
	}
	b, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return err
	}

	err = checkAndOverWriteFile(filePath, b)

	return err
}

func checkAndOverWriteFile(filePath string, b []byte) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// #nosec G306
		if err = ioutil.WriteFile(filePath, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func prepareDaprInstallDir(daprBinDir string) error {
	err := os.MkdirAll(daprBinDir, 0o777)
	if err != nil {
		return err
	}

	err = os.Chmod(daprBinDir, 0o777)
	if err != nil {
		return err
	}

	return nil
}

func archiveExt() string {
	ext := "tar.gz"
	if runtime.GOOS == daprWindowsOS {
		ext = "zip"
	}

	return ext
}

func downloadBinary(dir, version, binaryFilePrefix, githubRepo string) (string, error) {
	fileURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/v%s/%s",
		cli_ver.DaprGitHubOrg,
		githubRepo,
		version,
		binaryName(binaryFilePrefix))

	return downloadFile(dir, fileURL)
}

func binaryName(binaryFilePrefix string) string {
	return fmt.Sprintf("%s_%s_%s.%s", binaryFilePrefix, runtime.GOOS, runtime.GOARCH, archiveExt())
}

func downloadFile(dir string, url string) (string, error) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]

	filepath := path.Join(dir, fileName)
	_, err := os.Stat(filepath)
	if os.IsExist(err) {
		return "", nil
	}
	client := http.Client{ //nolint:exhaustruct
		Timeout: 0,
		Transport: &http.Transport{ //nolint:exhaustruct
			Dial: (&net.Dialer{ //nolint:exhaustruct
				Timeout: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("version not found from url: %s", url)
	} else if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed with %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = copyWithTimeout(context.Background(), out, resp.Body)
	if err != nil {
		return "", err
	}

	return filepath, nil
}

/*!
See: https://github.com/microsoft/vscode-winsta11er/blob/4b42060da64aea6f47adebe1dd654980ed87a046/common/common.go
Copyright (c) Microsoft Corporation. All rights reserved. Licensed under the MIT License.
*/
func copyWithTimeout(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	// Every 5 seconds, ensure at least 200 bytes (40 bytes/second average) are read.
	interval := 5
	minCopyBytes := int64(200)
	prevWritten := int64(0)
	written := int64(0)

	done := make(chan error)
	mu := sync.Mutex{}
	t := time.NewTicker(time.Duration(interval) * time.Second)
	defer t.Stop()

	// Read the stream, 32KB at a time.
	go func() {
		var (
			writeErr, readErr     error
			writeBytes, readBytes int
			buf                   = make([]byte, 32<<10)
		)
		for {
			readBytes, readErr = src.Read(buf)
			if readBytes > 0 {
				// Write to disk and update the number of bytes written.
				writeBytes, writeErr = dst.Write(buf[0:readBytes])
				mu.Lock()
				written += int64(writeBytes)
				mu.Unlock()
				if writeErr != nil {
					done <- writeErr
					return
				}
			}
			if readErr != nil {
				// If error is EOF, means we read the entire file, so don't consider that as error.
				if !errors.Is(readErr, io.EOF) {
					done <- readErr
					return
				}

				// No error.
				done <- nil
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		case <-t.C:
			mu.Lock()
			if written < prevWritten+minCopyBytes {
				mu.Unlock()
				return written, fmt.Errorf("stream stalled: received %d bytes over the last %d seconds", written, interval)
			}
			prevWritten = written
			mu.Unlock()
		case e := <-done:
			return written, e
		}
	}
}

// getPlacementImageName returns the resolved placement image name for online `dapr init`.
// It can either be resolved to the image-registry if given, otherwise GitHub container registry if
// selected or fallback to Docker Hub.
func getPlacementImageName(imageInfo daprImageInfo, info initInfo) (string, error) {
	image, err := resolveImageURI(imageInfo)
	if err != nil {
		return "", err
	}

	image = getPlacementImageWithTag(image, info.runtimeVersion)

	// if default registry is GHCR and the image is not available in or cannot be pulled from GHCR
	// fallback to using dockerhub.
	if useGHCR(imageInfo, info.fromDir) && !tryPullImage(image) {
		print.InfoStatusEvent(os.Stdout, "Placement image not found in Github container registry, pulling it from Docker Hub")
		image = getPlacementImageWithTag(daprDockerImageName, info.runtimeVersion)
	}
	return image, nil
}

func getPlacementImageWithTag(name, version string) string {
	if version == latestVersion {
		return name
	}
	return fmt.Sprintf("%s:%s", name, version)
}

// useGHCR returns true iff default registry is set as GHCR and --image-registry and --from-dir flags are not set.
// TODO: We may want to remove this logic completely after next couple of releases.
func useGHCR(imageInfo daprImageInfo, fromDir string) bool {
	if imageInfo.imageRegistryURL != "" || fromDir != "" {
		return false
	}
	return imageInfo.imageRegistryName == githubContainerRegistryName
}

func resolveImageURI(imageInfo daprImageInfo) (string, error) {
	if strings.TrimSpace(imageInfo.imageRegistryURL) != "" {
		if imageInfo.imageRegistryURL == ghcrURI || imageInfo.imageRegistryURL == "docker.io" {
			err := fmt.Errorf("flag --image-registry not set correctly. It cannot be %q or \"docker.io\"", ghcrURI)
			return "", err
		}
		return fmt.Sprintf(privateRegTemplateString, imageInfo.imageRegistryURL, imageInfo.ghcrImageName), nil
	}
	switch imageInfo.imageRegistryName {
	case dockerContainerRegistryName:
		return imageInfo.dockerHubImageName, nil
	case githubContainerRegistryName:
		return fmt.Sprintf("%s/%s", ghcrURI, imageInfo.ghcrImageName), nil
	default:
		return "", fmt.Errorf("imageRegistryName not set correctly %s", imageInfo.imageRegistryName)
	}
}

// setAirGapInit is used to set the bool value.
func setAirGapInit(fromDir string) {
	// mostly this is used for unit testing aprat from one use in Init() function.
	isAirGapInit = (len(strings.TrimSpace(fromDir)) != 0)
}
