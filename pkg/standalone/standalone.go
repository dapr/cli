// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	path_filepath "path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"

	"github.com/dapr/cli/pkg/print"
	cli_ver "github.com/dapr/cli/pkg/version"
	"github.com/dapr/cli/utils"
)

const (
	daprDockerImageName        = "daprio/dapr"
	daprRuntimeFilePrefix      = "daprd"
	dashboardFilePrefix        = "dashboard"
	placementServiceFilePrefix = "placement"
	daprWindowsOS              = "windows"
	latestVersion              = "latest"
	daprDefaultHost            = "localhost"
	pubSubYamlFileName         = "pubsub.yaml"
	stateStoreYamlFileName     = "statestore.yaml"

	// DaprPlacementContainerName is the container name of placement service.
	DaprPlacementContainerName = "dapr_placement"
	// DaprRedisContainerName is the container name of redis.
	DaprRedisContainerName = "dapr_redis"
	// DaprZipkinContainerName is the container name of zipkin.
	DaprZipkinContainerName = "dapr_zipkin"

	errInstallTemplate = "please run `dapr uninstall` first before running `dapr init`"
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
func Init(runtimeVersion, dashboardVersion string, dockerNetwork string, slimMode bool) error {
	if !slimMode {
		dockerInstalled := utils.IsDockerInstalled()
		if !dockerInstalled {
			return errors.New("could not connect to Docker. Docker may not be installed or running")
		}
	}

	if runtimeVersion == latestVersion {
		var err error
		runtimeVersion, err = cli_ver.GetDaprVersion()
		if err != nil {
			return fmt.Errorf("cannot get the latest release version: '%s'. Try specifying --runtime-version=<desired_version>", err)
		}
	}

	print.InfoStatusEvent(os.Stdout, "Installing runtime version %s", runtimeVersion)

	if dashboardVersion == latestVersion {
		var err error
		dashboardVersion, err = cli_ver.GetDashboardVersion()
		if err != nil {
			print.WarningStatusEvent(os.Stdout, "cannot get the latest dashboard version: '%s'. Try specifying --dashboard-version=<desired_version>", err)
			print.WarningStatusEvent(os.Stdout, "continuing, but dashboard will be unavailable")
		}
	}

	daprBinDir := defaultDaprBinPath()
	err := prepareDaprInstallDir(daprBinDir)
	if err != nil {
		return err
	}

	// confirm if installation is required.
	if ok, er := isBinaryInstallationRequired(daprRuntimeFilePrefix, daprBinDir); !ok {
		return er
	}

	var wg sync.WaitGroup
	errorChan := make(chan error)
	initSteps := []func(*sync.WaitGroup, chan<- error, string, string, string){}
	if slimMode {
		// Install 3 binaries in slim mode: daprd, dashboard, placement
		wg.Add(3)
		initSteps = append(initSteps, createSlimConfiguration)
	} else if dashboardVersion != "" {
		// Install 2 binaries: daprd, dashboard
		wg.Add(2)
		initSteps = append(initSteps, createComponentsAndConfiguration, runPlacementService, runRedis, runZipkin)
	} else {
		// Install 1 binaries: daprd
		wg.Add(1)
		initSteps = append(initSteps, createComponentsAndConfiguration, runPlacementService, runRedis, runZipkin)
	}

	// Init other configurations, containers
	wg.Add(len(initSteps))

	msg := "Downloading binaries and setting up components..."
	stopSpinning := print.Spinner(os.Stdout, msg)
	defer stopSpinning(print.Failure)

	// Make default components directory
	err = makeDefaultComponentsDir()
	if err != nil {
		return err
	}

	// Initialize daprd binary
	go installBinary(&wg, errorChan, daprBinDir, runtimeVersion, daprRuntimeFilePrefix, dockerNetwork, cli_ver.DaprGitHubRepo)

	// Initialize dashboard binary
	if dashboardVersion != "" {
		go installBinary(&wg, errorChan, daprBinDir, dashboardVersion, dashboardFilePrefix, dockerNetwork, cli_ver.DashboardGitHubRepo)
	}

	if slimMode {
		// Initialize placement binary only on slim install
		go installBinary(&wg, errorChan, daprBinDir, runtimeVersion, placementServiceFilePrefix, dockerNetwork, cli_ver.DaprGitHubRepo)
	}

	for _, step := range initSteps {
		// Run init on the configurations and containers
		go step(&wg, errorChan, daprBinDir, runtimeVersion, dockerNetwork)
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
	print.SuccessStatusEvent(os.Stdout, msg)
	print.InfoStatusEvent(os.Stdout, "%s binary has been installed to %s.", daprRuntimeFilePrefix, daprBinDir)
	if slimMode {
		// Print info on placement binary only on slim install
		print.InfoStatusEvent(os.Stdout, "%s binary has been installed to %s.", placementServiceFilePrefix, daprBinDir)
	} else {
		dockerContainerNames := []string{DaprPlacementContainerName, DaprRedisContainerName, DaprZipkinContainerName}
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

func prepareDaprInstallDir(daprBinDir string) error {
	err := os.MkdirAll(daprBinDir, 0777)
	if err != nil {
		return err
	}

	err = os.Chmod(daprBinDir, 0777)
	if err != nil {
		return err
	}

	return nil
}

func runZipkin(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string) {
	defer wg.Done()

	zipkinContainerName := utils.CreateContainerName(DaprZipkinContainerName, dockerNetwork)

	exists, err := confirmContainerIsRunningOrExists(zipkinContainerName, false)
	if err != nil {
		errorChan <- err
		return
	}
	args := []string{}

	if exists {
		// do not create container again if it exists
		args = append(args, "start", zipkinContainerName)
	} else {
		args = append(args,
			"run",
			"--name", zipkinContainerName,
			"--restart", "always",
			"-d",
		)

		if dockerNetwork != "" {
			args = append(
				args,
				"--network", dockerNetwork,
				"--network-alias", DaprZipkinContainerName)
		} else {
			args = append(
				args,
				"-p", "9411:9411")
		}

		args = append(args, "openzipkin/zipkin")
	}
	_, err = utils.RunCmdAndWait("docker", args...)

	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseDockerError("Zipkin tracing", err)
		} else {
			errorChan <- fmt.Errorf("docker %s failed with: %v", args, err)
		}
		return
	}
	errorChan <- nil
}

func runRedis(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string) {
	defer wg.Done()

	redisContainerName := utils.CreateContainerName(DaprRedisContainerName, dockerNetwork)

	exists, err := confirmContainerIsRunningOrExists(redisContainerName, false)
	if err != nil {
		errorChan <- err
		return
	}
	args := []string{}

	if exists {
		// do not create container again if it exists
		args = append(args, "start", redisContainerName)
	} else {
		args = append(args,
			"run",
			"--name", redisContainerName,
			"--restart", "always",
			"-d",
		)

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
	}
	_, err = utils.RunCmdAndWait("docker", args...)

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

func isContainerRunError(err error) bool {
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode := exitError.ExitCode()
		return exitCode == 125
	}
	return false
}

func runPlacementService(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string) {
	defer wg.Done()
	placementContainerName := utils.CreateContainerName(DaprPlacementContainerName, dockerNetwork)

	image := fmt.Sprintf("%s:%s", daprDockerImageName, version)

	// Use only image for latest version
	if version == latestVersion {
		image = daprDockerImageName
	}

	exists, err := confirmContainerIsRunningOrExists(placementContainerName, false)

	if err != nil {
		errorChan <- err
		return
	} else if exists {
		errorChan <- fmt.Errorf("%s container exists or is running. %s", placementContainerName, errInstallTemplate)
		return
	}

	args := []string{
		"run",
		"--name", placementContainerName,
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

	_, err = utils.RunCmdAndWait("docker", args...)

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

func moveDashboardFiles(extractedFilePath string, dir string) (string, error) {
	// Move /release/os/web directory to /web
	oldPath := path_filepath.Join(path_filepath.Dir(extractedFilePath), "web")
	newPath := path_filepath.Join(dir, "web")
	err := os.Rename(oldPath, newPath)
	if err != nil {
		err = fmt.Errorf("failed to move dashboard files: %s", err)
		return "", err
	}

	// Move binary from /release/<os>/web/dashboard(.exe) to /dashboard(.exe)
	err = os.Rename(extractedFilePath, path_filepath.Join(dir, path_filepath.Base(extractedFilePath)))
	if err != nil {
		err = fmt.Errorf("error moving %s binary to path: %s", path_filepath.Base(extractedFilePath), err)
		return "", err
	}

	// Change the extracted binary file path to reflect the move above
	extractedFilePath = path_filepath.Join(dir, path_filepath.Base(extractedFilePath))

	// Remove the now-empty 'release' directory
	err = os.RemoveAll(path_filepath.Join(dir, "release"))
	if err != nil {
		err = fmt.Errorf("error moving dashboard files: %s", err)
		return "", err
	}

	return extractedFilePath, nil
}

func installBinary(wg *sync.WaitGroup, errorChan chan<- error, dir, version, binaryFilePrefix string, dockerNetwork string, githubRepo string) {
	defer wg.Done()

	archiveExt := "tar.gz"

	if runtime.GOOS == daprWindowsOS {
		archiveExt = "zip"
	}

	fileURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/v%s/%s_%s_%s.%s",
		cli_ver.DaprGitHubOrg,
		githubRepo,
		version,
		binaryFilePrefix,
		runtime.GOOS,
		runtime.GOARCH,
		archiveExt)

	filepath, err := downloadFile(dir, fileURL)
	if err != nil {
		errorChan <- fmt.Errorf("error downloading %s binary: %s", binaryFilePrefix, err)
		return
	}

	extractedFilePath := ""

	if archiveExt == "zip" {
		extractedFilePath, err = unzip(filepath, dir, binaryFilePrefix)
	} else {
		extractedFilePath, err = untar(filepath, dir, binaryFilePrefix)
	}

	if err != nil {
		errorChan <- fmt.Errorf("error extracting %s binary: %s", binaryFilePrefix, err)
		return
	}
	err = os.Remove(filepath)

	if err != nil {
		errorChan <- fmt.Errorf("failed to remove archive: %s", err)
		return
	}

	if binaryFilePrefix == "dashboard" {
		extractedFilePath, err = moveDashboardFiles(extractedFilePath, dir)
		if err != nil {
			errorChan <- err
			return
		}
	}

	binaryPath, err := moveFileToPath(extractedFilePath, dir)
	if err != nil {
		errorChan <- fmt.Errorf("error moving %s binary to path: %s", binaryFilePrefix, err)
		return
	}

	err = makeExecutable(binaryPath)
	if err != nil {
		errorChan <- fmt.Errorf("error making %s binary executable: %s", binaryFilePrefix, err)
		return
	}

	errorChan <- nil
}

func createComponentsAndConfiguration(wg *sync.WaitGroup, errorChan chan<- error, _, _ string, dockerNetwork string) {
	defer wg.Done()

	redisHost := daprDefaultHost
	zipkinHost := daprDefaultHost
	if dockerNetwork != "" {
		// Default to network scoped alias of the container names when a dockerNetwork is specified.
		redisHost = DaprRedisContainerName
		zipkinHost = DaprZipkinContainerName
	}
	var err error

	// Make default components directory
	componentsDir := DefaultComponentsDirPath()

	err = createRedisPubSub(redisHost, componentsDir)
	if err != nil {
		errorChan <- fmt.Errorf("error creating redis pubsub component file: %s", err)
		return
	}
	err = createRedisStateStore(redisHost, componentsDir)
	if err != nil {
		errorChan <- fmt.Errorf("error creating redis statestore component file: %s", err)
		return
	}
	err = createDefaultConfiguration(zipkinHost, DefaultConfigFilePath())
	if err != nil {
		errorChan <- fmt.Errorf("error creating default configuration file: %s", err)
		return
	}
}

func createSlimConfiguration(wg *sync.WaitGroup, errorChan chan<- error, _, _ string, _ string) {
	defer wg.Done()

	// For --slim we pass empty string so that we do not configure zipkin.
	err := createDefaultConfiguration("", DefaultConfigFilePath())
	if err != nil {
		errorChan <- fmt.Errorf("error creating default configuration file: %s", err)
		return
	}
}

func makeDefaultComponentsDir() error {
	// Make default components directory
	componentsDir := DefaultComponentsDirPath()
	_, err := os.Stat(componentsDir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(componentsDir, 0755)
		if errDir != nil {
			return fmt.Errorf("error creating default components folder: %s", errDir)
		}
	}

	os.Chmod(componentsDir, 0777)
	return nil
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

// https://github.com/snyk/zip-slip-vulnerability, fixes gosec G305
func sanitizeExtractPath(destination string, filePath string) (string, error) {
	destpath := path_filepath.Join(destination, filePath)
	if !strings.HasPrefix(destpath, path_filepath.Clean(destination)+string(os.PathSeparator)) {
		return "", fmt.Errorf("%s: illegal file path", filePath)
	}
	return destpath, nil
}

func unzip(filepath, targetDir, binaryFilePrefix string) (string, error) {
	r, err := zip.OpenReader(filepath)
	if err != nil {
		return "", err
	}
	defer r.Close()

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

func untar(filepath, targetDir, binaryFilePrefix string) (string, error) {
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

	foundBinary := ""
	for {
		header, err := tr.Next()

		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		} else if header == nil {
			continue
		}

		// untar all files in archive
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

		// If the found file is the binary that we want to find, save it and return later
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
	if err = ioutil.WriteFile(destFilePath, input, 0644); err != nil {
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
		return "", fmt.Errorf("version not found from url: %s", url)
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
		defaultConfig.Spec.Tracing.Zipkin.EndpointAddress = fmt.Sprintf("http://%s:9411/api/v2/spans", zipkinHost)
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
		if err = ioutil.WriteFile(filePath, b, 0644); err != nil {
			return err
		}
	}
	return nil
}
