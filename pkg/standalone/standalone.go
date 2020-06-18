// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
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
	"os/user"
	"path"
	path_filepath "path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"

	"github.com/briandowns/spinner"
	"github.com/dapr/cli/pkg/print"
	cli_ver "github.com/dapr/cli/pkg/version"
	"github.com/dapr/cli/utils"
)

const (
	daprDockerImageName               = "daprio/dapr"
	daprRuntimeFilePrefix             = "daprd"
	daprWindowsOS                     = "windows"
	daprLatestVersion                 = "latest"
	daprDefaultLinuxAndMacInstallPath = "/usr/local/bin"
	daprDefaultWindowsInstallPath     = "c:\\dapr"
	daprDefaultHost                   = "localhost"
	pubSubYamlFileName                = "pubsub.yaml"
	stateStoreYamlFileName            = "statestore.yaml"
	zipkinYamlFileName                = "zipkin.yaml"

	// DaprPlacementContainerName is the container name of placement service
	DaprPlacementContainerName = "dapr_placement"
	// DaprRedisContainerName is the container name of redis
	DaprRedisContainerName = "dapr_redis"
	// DaprZipkinContainerName is the container name of zipkin
	DaprZipkinContainerName = "dapr_zipkin"
)

type configuration struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Tracing struct {
			SamplingRate string `yaml:"samplingRate"`
		} `yaml:"tracing"`
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
		Metadata []componentMetadataItem `yaml:"metadata"`
	} `yaml:"spec"`
}

type componentMetadataItem struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func isInstallationRequired(installLocation, requestedVersion string) bool {
	var destDir string

	// if specified using --install-path
	if installLocation != "" {
		destDir = installLocation
	} else {
		if runtime.GOOS == daprWindowsOS {
			destDir = daprDefaultWindowsInstallPath
		} else {
			destDir = daprDefaultLinuxAndMacInstallPath
		}
	}

	// e.g. /usr/local/bin/daprd or c:\dapr, which are the defaults unless overridden by "installLocation"
	daprdBinaryPath := path_filepath.Join(destDir, daprRuntimeFilePrefix)

	// first time install?
	_, err := os.Stat(daprdBinaryPath)
	if os.IsNotExist(err) {
		return true
	}

	var msg string

	// what's the installed version?
	v, err := utils.RunCmdAndWait(daprdBinaryPath, "--version")
	if err != nil {
		msg = fmt.Sprintf("unable to determine installed Dapr version at %s. installation will continue", destDir)
		fmt.Println(msg)
		return true
	}
	installedVersion := strings.TrimSpace(v)

	// "latest" version requested. need to check the corresponding version
	if requestedVersion == daprLatestVersion {
		latestVersion, err := cli_ver.GetLatestRelease(cli_ver.DaprGitHubOrg, cli_ver.DaprGitHubRepo)
		if err != nil {
			msg = fmt.Sprintf("latest Dapr version information could not be found - %s", err.Error())
			fmt.Println(msg)
			return false
		}
		latestVersion = latestVersion[1:]
		if installedVersion == latestVersion {
			msg = fmt.Sprintf("required version %s is the same as installed version at %s", latestVersion, destDir)
			fmt.Println(msg)
			return false
		}
	}

	// if daprd exists, need to confirm if the intended version is same as the current one
	if installedVersion == requestedVersion {
		msg = fmt.Sprintf("required version %s is the same as installed version at %s", requestedVersion, destDir)
		fmt.Println(msg)
		return false
	}

	return true
}

// Init installs Dapr on a local machine using the supplied runtimeVersion.
func Init(runtimeVersion string, dockerNetwork string, installLocation string, redisHost string) error {
	dockerInstalled := utils.IsDockerInstalled()
	if !dockerInstalled {
		return errors.New("could not connect to Docker. Docker may not be installed or running")
	}

	downloadDest, err := getDownloadDest(installLocation)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errorChan := make(chan error)

	initSteps := []func(*sync.WaitGroup, chan<- error, string, string, string, string, string){}
	initSteps = append(initSteps, installDaprBinary, createComponentsAndConfiguration, runPlacementService, runRedis, runZipkin)
	dockerContainerNames := []string{DaprPlacementContainerName, DaprRedisContainerName, DaprZipkinContainerName}

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
		go step(&wg, errorChan, downloadDest, runtimeVersion, dockerNetwork, installLocation, redisHost)
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
	}

	print.SuccessStatusEvent(os.Stdout, msg)
	print.InfoStatusEvent(os.Stdout, "%s binary has been installed.\n", daprRuntimeFilePrefix)
	for _, container := range dockerContainerNames {
		err = confirmContainerIsRunning(utils.CreateContainerName(container, dockerNetwork))
		if err != nil {
			return err
		}
		print.InfoStatusEvent(os.Stdout, "%s container is running.\n", container)
	}
	print.InfoStatusEvent(os.Stdout, "Use `docker ps` to check running containers.\n")
	return nil
}

func getDownloadDest(installLocation string) (string, error) {
	p := ""

	// use the install location passed in for Windows.  This can't
	// be done for other environments because the install location default to a privileged dir: /usr/local/bin
	if runtime.GOOS == daprWindowsOS {
		if installLocation == "" {
			p = daprDefaultWindowsInstallPath
		} else {
			p = installLocation
		}
	} else {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		p = path.Join(usr.HomeDir, ".dapr")
	}

	err := os.MkdirAll(p, 0777)
	if err != nil {
		return "", err
	}

	err = os.Chmod(p, 0777)
	if err != nil {
		return "", err
	}

	return p, nil
}

// installLocation is not used, but it is present because it's required to fit the initSteps func above.
// If the number of args increases more, we may consider passing in a struct instead of individual args.

func runZipkin(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string, installLocation string, redisHost string) {
	defer wg.Done()

	args := []string{
		"run",
		"--name", utils.CreateContainerName(DaprZipkinContainerName, dockerNetwork),
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
			"-p", "9411:9411")
	}

	args = append(args, "openzipkin/zipkin")
	_, err := utils.RunCmdAndWait("docker", args...)

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

func runRedis(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string, installLocation string, redisHost string) {
	defer wg.Done()

	if redisHost != daprDefaultHost {
		// A non-default Redis host is specified. No need to start the redis container
		fmt.Printf("You have specified redis-host: %s. Make sure you have a redis server running there.\n", redisHost)
		return
	}

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

func runPlacementService(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string, installLocation string, _ string) {
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

func installDaprBinary(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string, installLocation string, _ string) {
	defer wg.Done()

	// confirm if installation is required
	if !isInstallationRequired(installLocation, version) {
		return
	}

	archiveExt := "tar.gz"
	if runtime.GOOS == daprWindowsOS {
		archiveExt = "zip"
	}

	if version == daprLatestVersion {
		var err error
		version, err = cli_ver.GetLatestRelease(cli_ver.DaprGitHubOrg, cli_ver.DaprGitHubRepo)
		if err != nil {
			errorChan <- fmt.Errorf("cannot get the latest release version: %s", err)
			return
		}
		version = version[1:]
	}

	daprURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/v%s/%s_%s_%s.%s",
		cli_ver.DaprGitHubOrg,
		cli_ver.DaprGitHubRepo,
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

func createComponentsAndConfiguration(wg *sync.WaitGroup, errorChan chan<- error, dir, version string, dockerNetwork string, installLocation string, redisHost string) {
	defer wg.Done()

	var err error

	// Make default components directory
	componentsDir := DefaultFolderPath(DefaultComponentsDirName)
	_, err = os.Stat(componentsDir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(componentsDir, 0755)
		if errDir != nil {
			errorChan <- fmt.Errorf("error creating default components folder: %s", errDir)
			return
		}
	}

	os.Chmod(componentsDir, 0777)

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
	err = createZipkinComponent(daprDefaultHost, componentsDir)
	if err != nil {
		errorChan <- fmt.Errorf("error creating zipkin component file: %s", err)
		return
	}
	err = createDefaultConfiguration(DefaultConfigFilePath())
	if err != nil {
		errorChan <- fmt.Errorf("error creating default configuration file: %s", err)
		return
	}
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

		// #nosec G110
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

			// #nosec G110
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

	if installLocation != "" {
		color.Set(color.FgYellow)
		fmt.Printf("\nDapr installed to %s, please run the following to add it to your path:\n", destDir)
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

func createRedisStateStore(redisHost string, componentsPath string) error {
	redisStore := component{
		APIVersion: "dapr.io/v1alpha1",
		Kind:       "Component",
	}

	redisStore.Metadata.Name = "statestore"
	redisStore.Spec.Type = "state.redis"
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

func createDefaultConfiguration(filePath string) error {
	defaultConfig := configuration{
		APIVersion: "dapr.io/v1alpha1",
		Kind:       "Configuration",
	}
	defaultConfig.Metadata.Name = "daprConfig"
	defaultConfig.Spec.Tracing.SamplingRate = "1"

	b, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return err
	}

	err = checkAndOverWriteFile(filePath, b)

	return err
}

func createZipkinComponent(zipkinHost string, componentsPath string) error {
	zipKinComponent := component{
		APIVersion: "dapr.io/v1alpha1",
		Kind:       "Component",
	}
	zipKinComponent.Metadata.Name = "zipkin"
	zipKinComponent.Spec.Type = "exporters.zipkin"
	zipKinComponent.Spec.Metadata = []componentMetadataItem{
		{
			Name:  "enabled",
			Value: "true",
		},
		{
			Name:  "exporterAddress",
			Value: fmt.Sprintf("http://%s:9411/api/v2/spans", zipkinHost),
		},
	}

	b, err := yaml.Marshal(&zipKinComponent)
	if err != nil {
		return err
	}

	filePath := path_filepath.Join(componentsPath, zipkinYamlFileName)
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
	} else {
		fmt.Printf("file %s exists in the default dapr path", filePath)
	}
	return nil
}
