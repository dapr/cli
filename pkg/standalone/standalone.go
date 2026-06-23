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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	path_filepath "path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"
	"github.com/fatih/color"
	"gopkg.in/yaml.v2"

	"github.com/dapr/cli/pkg/print"
	cli_ver "github.com/dapr/cli/pkg/version"
	"github.com/dapr/cli/utils"
	"github.com/dapr/dapr/pkg/sentry/server/ca/bundle"
)

const (
	daprRuntimeFilePrefix      = "daprd"
	placementServiceFilePrefix = "placement"
	schedulerServiceFilePrefix = "scheduler"

	daprWindowsOS = "windows"

	latestVersion   = "latest"
	daprDefaultHost = "localhost"

	pubSubYamlFileName     = "pubsub.yaml"
	stateStoreYamlFileName = "statestore.yaml"

	// accepted DAPR_DEFAULT_IMAGE_REGISTRY values.
	dockerContainerRegistryName = "dockerhub"
	githubContainerRegistryName = "ghcr"

	// used when DAPR_DEFAULT_IMAGE_REGISTRY is not set.
	daprDockerImageName       = "docker.io/daprio/dapr"
	redisDockerImageName      = "docker.io/redis:6"
	redisStackDockerImageName = "docker.io/redis/redis-stack-server:7.2.0-v19"
	zipkinDockerImageName     = "docker.io/openzipkin/zipkin"

	// used when DAPR_DEFAULT_IMAGE_REGISTRY is set as GHCR.
	dockerURI = "docker.io"
	ghcrURI   = "ghcr.io"

	// used when DAPR_DEFAULT_IMAGE_REGISTRY is set as GHCR or image-registry flag is set.
	daprGhcrImageName       = "dapr/dapr"
	redisGhcrImageName      = "dapr/3rdparty/redis:6"
	redisStackGhcrImageName = "dapr/3rdparty/redis-stack-server:7.2.0-v19"
	zipkinGhcrImageName     = "dapr/3rdparty/zipkin"

	// DaprPlacementContainerName is the container name of placement service.
	DaprPlacementContainerName = "dapr_placement"
	// DaprSchedulerContainerName is the container name of scheduler service.
	DaprSchedulerContainerName = "dapr_scheduler"
	// DaprSentryContainerName is the container name of sentry service.
	DaprSentryContainerName = "dapr_sentry"
	// DaprRedisContainerName is the container name of redis.
	DaprRedisContainerName = "dapr_redis"
	// DaprZipkinContainerName is the container name of zipkin.
	DaprZipkinContainerName = "dapr_zipkin"

	errInstallTemplate = "please run `dapr uninstall` first before running `dapr init`"

	healthPort = 58080
	metricPort = 59090

	schedulerHealthPort = 58081
	schedulerMetricPort = 59091
	schedulerEtcdPort   = 2379

	sentryGRPCPort              = 50001
	sentryHealthPort            = 58082
	sentryMetricPort            = 59092
	sentryConfigContainerPath   = "/etc/dapr/config.yaml"
	sentryStandaloneMode        = "standalone"

	defaultTrustDomain = "cluster.local"
	trustAnchorsFile   = "ca.crt"
	issuerCertFile     = "issuer.crt"
	issuerKeyFile      = "issuer.key"

	daprVersionsWithScheduler = ">= 1.14.x"
)

var (
	defaultImageRegistryName string
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
		MTLS struct {
			Enabled bool `yaml:"enabled,omitempty"`
		} `yaml:"mtls,omitempty"`
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
	fromDir                            string
	installDir                         string
	bundleDet                          *bundleDetails
	slimMode                           bool
	enableMTLS                         bool
	runtimeVersion                     string
	dockerNetwork                      string
	imageRegistryURL                   string
	containerRuntime                   string
	imageVariant                       string
	schedulerVolume                    *string
	schedulerOverrideBroadcastHostPort *string
	redisStack                         bool
}

// InitOptions configures a standalone Dapr initialization.
type InitOptions struct {
	RuntimeVersion                     string
	DockerNetwork                      string
	SlimMode                           bool
	EnableMTLS                         bool
	ImageRegistryURL                   string
	FromDir                            string
	ContainerRuntime                   string
	ImageVariant                       string
	DaprInstallPath                    string
	SchedulerVolume                    *string
	SchedulerOverrideBroadcastHostPort *string
	RedisStack                         bool
}

type daprImageInfo struct {
	ghcrImageName      string
	dockerHubImageName string
	imageRegistryURL   string
	imageRegistryName  string
}

// Check if the previous version is already installed.
func isBinaryInstallationRequired(binaryFilePrefix, binInstallDir string) (bool, error) {
	binaryPath := binaryFilePathWithDir(binInstallDir, binaryFilePrefix)

	// first time install?
	_, err := os.Stat(binaryPath)
	if !os.IsNotExist(err) {
		return false, fmt.Errorf("%s %w, %s", binaryPath, os.ErrExist, errInstallTemplate)
	}
	return true, nil
}

// isSchedulerIncluded returns true if scheduler is included a given version for Dapr.
func isSchedulerIncluded(runtimeVersion string) (bool, error) {
	c, err := semver.NewConstraint(daprVersionsWithScheduler)
	if err != nil {
		return false, err
	}

	v, err := semver.NewVersion(runtimeVersion)
	if err != nil {
		return false, err
	}

	vNoPrerelease, err := v.SetPrerelease("")
	if err != nil {
		return false, err
	}
	return c.Check(&vNoPrerelease), nil
}

// Init installs Dapr on a local machine using the supplied runtimeVersion.
func Init(opts InitOptions) error {
	var err error
	var bundleDet bundleDetails
	runtimeVersion := opts.RuntimeVersion
	dockerNetwork := opts.DockerNetwork
	slimMode := opts.SlimMode
	enableMTLS := opts.EnableMTLS
	imageRegistryURL := opts.ImageRegistryURL
	fromDir := opts.FromDir
	containerRuntime := opts.ContainerRuntime
	imageVariant := opts.ImageVariant
	daprInstallPath := opts.DaprInstallPath
	schedulerVolume := opts.SchedulerVolume
	schedulerOverrideBroadcastHostPort := opts.SchedulerOverrideBroadcastHostPort
	redisStack := opts.RedisStack

	if enableMTLS && slimMode {
		return fmt.Errorf("--enable-mtls is not supported with --slim mode")
	}

	containerRuntime = strings.TrimSpace(containerRuntime)
	daprInstallPath = strings.TrimSpace(daprInstallPath)
	// AirGap init flow is true when fromDir var is set i.e. --from-dir flag has value.
	fromDir = strings.TrimSpace(fromDir)
	setAirGapInit(fromDir)
	if !slimMode {
		// If --slim installation is not requested, check if docker is installed.
		containerRuntimeAvailable := utils.IsContainerRuntimeInstalled(containerRuntime)
		if !containerRuntimeAvailable {
			return fmt.Errorf("could not connect to %s. %s may not be installed or running", containerRuntime, containerRuntime)
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

	// Determine the effective registry URL for version resolution.
	// When no custom registry is provided, use the default registry
	// (GHCR or Docker Hub) so we query tags from the same registry
	// that will be used for pulling images.
	effectiveRegistryURL := imageRegistryURL
	if effectiveRegistryURL == "" && defaultImageRegistryName == githubContainerRegistryName {
		effectiveRegistryURL = ghcrURI
	}

	if runtimeVersion == latestVersion && !isAirGapInit {
		runtimeVersion, err = cli_ver.GetLatestVersion(cli_ver.DaprImageRef(effectiveRegistryURL))
		if err != nil {
			return fmt.Errorf("cannot get the latest release version: '%w'. Try specifying --runtime-version=<desired_version>", err)
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

		// Set runtime version from the bundle details parsed.
		runtimeVersion = *bundleDet.RuntimeVersion
	}

	// At this point the runtimeVersion variable is parsed either from the details file if --fromDir is specified or
	// got from running the command cli_ver.GetRuntimeVersion().

	// After this point runtimeVersion will not be latest string but rather actual version.

	print.InfoStatusEvent(os.Stdout, "Installing runtime version %s", runtimeVersion)

	installDir, err := GetDaprRuntimePath(daprInstallPath)
	if err != nil {
		return err
	}
	daprBinDir := getDaprBinPath(installDir)
	err = prepareDaprInstallDir(daprBinDir)
	if err != nil {
		return err
	}

	// confirm if installation is required.
	if ok, er := isBinaryInstallationRequired(daprRuntimeFilePrefix, daprBinDir); !ok {
		return er
	}

	prepSteps := []func(*sync.WaitGroup, chan<- error, initInfo){
		createSlimConfiguration,
		createComponentsAndConfiguration,
		generateCertsForMTLS,
		installDaprRuntime,
		installPlacement,
		installScheduler,
	}
	containerSteps := []func(*sync.WaitGroup, chan<- error, initInfo){
		runPlacementService,
		runSchedulerService,
		runRedis,
		runZipkin,
		runSentryService,
	}

	msg := "Downloading binaries and setting up components..."
	if isAirGapInit {
		msg = "Extracting binaries and setting up components..."
	}
	stopSpinning := print.Spinner(os.Stdout, "%s", msg)
	defer stopSpinning(print.Failure)

	// Make default components directory.
	err = makeDefaultComponentsDir(installDir)
	if err != nil {
		return err
	}

	info := initInfo{
		// values in bundleDet can be nil if fromDir is empty, so must be used in conjunction with fromDir.
		bundleDet:                          &bundleDet,
		fromDir:                            fromDir,
		installDir:                         installDir,
		slimMode:                           slimMode,
		enableMTLS:                         enableMTLS,
		runtimeVersion:                     runtimeVersion,
		dockerNetwork:                      dockerNetwork,
		imageRegistryURL:                   imageRegistryURL,
		containerRuntime:                   containerRuntime,
		imageVariant:                       imageVariant,
		schedulerVolume:                    schedulerVolume,
		schedulerOverrideBroadcastHostPort: schedulerOverrideBroadcastHostPort,
		redisStack:                         redisStack,
	}
	if enableMTLS {
		if err := runParallelInitSteps(prepSteps, info); err != nil {
			return err
		}
		if err := runSentryServiceInternal(info); err != nil {
			return err
		}
		if err := runParallelInitSteps(containerSteps, info); err != nil {
			return err
		}
	} else {
		initSteps := append(prepSteps, containerSteps...)
		var wg sync.WaitGroup
		errorChan := make(chan error)
		wg.Add(len(initSteps))
		for _, step := range initSteps {
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
	}

	stopSpinning(print.Success)

	msg = "Downloaded binaries and completed components set up."
	if isAirGapInit {
		msg = "Extracted binaries and completed components set up."
	}
	print.SuccessStatusEvent(os.Stdout, "%s", msg)
	print.InfoStatusEvent(os.Stdout, "%s binary has been installed to %s.", daprRuntimeFilePrefix, daprBinDir)
	if slimMode {
		// Print info on placement binary only on slim install.
		print.InfoStatusEvent(os.Stdout, "%s binary has been installed to %s.", placementServiceFilePrefix, daprBinDir)
		print.InfoStatusEvent(os.Stdout, "%s binary has been installed to %s.", schedulerServiceFilePrefix, daprBinDir)
	} else {
		runtimeCmd := utils.GetContainerRuntimeCmd(info.containerRuntime)
		dockerContainerNames := []string{DaprPlacementContainerName, DaprRedisContainerName, DaprZipkinContainerName}
		// Skip redis and zipkin in local installation mode.
		if isAirGapInit {
			dockerContainerNames = []string{DaprPlacementContainerName}
		}
		hasScheduler, err := isSchedulerIncluded(info.runtimeVersion)
		if err == nil && hasScheduler {
			dockerContainerNames = append(dockerContainerNames, DaprSchedulerContainerName)
		}
		if info.enableMTLS {
			dockerContainerNames = append(dockerContainerNames, DaprSentryContainerName)
		}
		for _, container := range dockerContainerNames {
			containerName := utils.CreateContainerName(container, dockerNetwork)
			ok, err := confirmContainerIsRunningOrExists(containerName, true, runtimeCmd)
			if err != nil {
				return err
			}
			if ok {
				print.InfoStatusEvent(os.Stdout, "%s container is running.", containerName)
			}
		}
		print.InfoStatusEvent(os.Stdout, "Use `%s ps` to check running containers.", runtimeCmd)
		if info.enableMTLS {
			sentryContainerName := utils.CreateContainerName(DaprSentryContainerName, dockerNetwork)
			ok, err := confirmContainerIsRunningOrExists(sentryContainerName, true, runtimeCmd)
			if err != nil {
				return err
			}
			if ok {
				print.InfoStatusEvent(os.Stdout, "Sentry running, mTLS enabled, trust domain: %s", defaultTrustDomain)
			}
		}
	}
	return nil
}

func runZipkin(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if info.slimMode || isAirGapInit {
		return
	}

	zipkinContainerName := utils.CreateContainerName(DaprZipkinContainerName, info.dockerNetwork)

	runtimeCmd := utils.GetContainerRuntimeCmd(info.containerRuntime)
	exists, err := confirmContainerIsRunningOrExists(zipkinContainerName, false, runtimeCmd)
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
	_, err = utils.RunCmdAndWait(runtimeCmd, args...)
	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseContainerRuntimeError("Zipkin tracing", err)
		} else {
			errorChan <- fmt.Errorf("%s %s failed with: %w", runtimeCmd, args, err)
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

	runtimeCmd := utils.GetContainerRuntimeCmd(info.containerRuntime)
	exists, err := confirmContainerIsRunningOrExists(redisContainerName, false, runtimeCmd)
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
		imageName, err = resolveImageURI(redisImageInfo(info.redisStack, info.imageRegistryURL, defaultImageRegistryName))
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
	_, err = utils.RunCmdAndWait(runtimeCmd, args...)
	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseContainerRuntimeError("Redis state store", err)
		} else {
			errorChan <- fmt.Errorf("%s %s failed with: %w", runtimeCmd, args, err)
		}
		return
	}
	errorChan <- nil
}

func redisImageInfo(redisStack bool, imageRegistryURL string, imageRegistryName string) daprImageInfo {
	if redisStack {
		return daprImageInfo{
			ghcrImageName:      redisStackGhcrImageName,
			dockerHubImageName: redisStackDockerImageName,
			imageRegistryURL:   imageRegistryURL,
			imageRegistryName:  imageRegistryName,
		}
	}

	return daprImageInfo{
		ghcrImageName:      redisGhcrImageName,
		dockerHubImageName: redisDockerImageName,
		imageRegistryURL:   imageRegistryURL,
		imageRegistryName:  imageRegistryName,
	}
}

func runPlacementService(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if info.slimMode {
		return
	}

	runtimeCmd := utils.GetContainerRuntimeCmd(info.containerRuntime)
	placementContainerName := utils.CreateContainerName(DaprPlacementContainerName, info.dockerNetwork)

	exists, err := confirmContainerIsRunningOrExists(placementContainerName, false, runtimeCmd)

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
		image = info.bundleDet.getDaprImageName()
		err = loadContainer(dir, info.bundleDet.getDaprImageFileName(), info.containerRuntime)
		if err != nil {
			errorChan <- err
			return
		}
	} else {
		// otherwise load the image from the specified repository.
		image, err = getDaprImageName(imgInfo, info)
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
			"-p", fmt.Sprintf("%v:50005", osPort),
			"-p", fmt.Sprintf("%v:8080", healthPort),
			"-p", fmt.Sprintf("%v:9090", metricPort),
		)
	}

	args = appendMTLSContainerRunArgs(args, info)
	args = append(args, image)
	if info.enableMTLS {
		args = append(args, mtlsControlPlaneServiceArgs(info.dockerNetwork)...)
	}

	_, err = utils.RunCmdAndWait(runtimeCmd, args...)
	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseContainerRuntimeError("placement service", err)
		} else {
			errorChan <- fmt.Errorf("%s %s failed with: %w", runtimeCmd, args, err)
		}
		return
	}
	errorChan <- nil
}

func runSchedulerService(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if info.slimMode {
		return
	}

	hasScheduler, err := isSchedulerIncluded(info.runtimeVersion)
	if err != nil {
		errorChan <- err
		return
	}
	if !hasScheduler {
		return
	}

	runtimeCmd := utils.GetContainerRuntimeCmd(info.containerRuntime)
	schedulerContainerName := utils.CreateContainerName(DaprSchedulerContainerName, info.dockerNetwork)

	exists, err := confirmContainerIsRunningOrExists(schedulerContainerName, false, runtimeCmd)

	if err != nil {
		errorChan <- err
		return
	} else if exists {
		errorChan <- fmt.Errorf("%s container exists or is running. %s", schedulerContainerName, errInstallTemplate)
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
		image = info.bundleDet.getDaprImageName()
		err = loadContainer(dir, info.bundleDet.getDaprImageFileName(), info.containerRuntime)
		if err != nil {
			errorChan <- err
			return
		}
	} else {
		// otherwise load the image from the specified repository.
		image, err = getDaprImageName(imgInfo, info)
		if err != nil {
			errorChan <- err
			return
		}
	}

	args := []string{
		"run",
		"--name", schedulerContainerName,
		"--restart", "always",
		"-d",
		"--entrypoint", "./scheduler",
	}
	if info.schedulerVolume != nil {
		// Don't touch this file location unless things start breaking.
		// In Docker, when Docker creates a volume and mounts that volume. Docker
		// assumes the file permissions of that directory if it exists in the container.
		// If that directory didn't exist in the container previously, then Docker sets
		// the permissions owned by root and not writeable.
		// We are lucky in that the Dapr containers have a world writeable directory at
		// /var/lock and can therefore mount the Docker volume here.
		// TODO: update the Dapr scheduler dockerfile to create a scheduler user id writeable
		// directory at /var/lib/dapr/scheduler, then update the path here.
		if strings.EqualFold(info.imageVariant, "mariner") {
			args = append(args, "--volume", *info.schedulerVolume+":/var/tmp")
		} else {
			args = append(args, "--volume", *info.schedulerVolume+":/var/lock")
		}
	}

	osPort := 50006
	if info.dockerNetwork != "" {
		args = append(args,
			"--network", info.dockerNetwork,
			"--network-alias", DaprSchedulerContainerName)
	} else {
		if runtime.GOOS == daprWindowsOS {
			osPort = 6060
		}

		args = append(args,
			"-p", fmt.Sprintf("%v:50006", osPort),
			"-p", fmt.Sprintf("%v:2379", schedulerEtcdPort),
			"-p", fmt.Sprintf("%v:8080", schedulerHealthPort),
			"-p", fmt.Sprintf("%v:9090", schedulerMetricPort),
		)
	}

	args = appendMTLSContainerRunArgs(args, info)

	if strings.EqualFold(info.imageVariant, "mariner") {
		args = append(args, image, "--etcd-data-dir=/var/tmp/dapr/scheduler")
	} else {
		args = append(args, image, "--etcd-data-dir=/var/lock/dapr/scheduler")
	}

	if schedulerOverrideHostPort(info) {
		if info.schedulerOverrideBroadcastHostPort != nil {
			args = append(args, "--override-broadcast-host-port="+*info.schedulerOverrideBroadcastHostPort)
		} else {
			args = append(args, fmt.Sprintf("--override-broadcast-host-port=localhost:%v", osPort))
		}
	}

	if schedulerEtcdClientListenAddress(info) {
		args = append(args, "--etcd-client-listen-address=0.0.0.0")
	}

	if info.enableMTLS {
		args = append(args, mtlsControlPlaneServiceArgs(info.dockerNetwork)...)
	}

	// On non-elevated Windows with WSL2 installed, verify the scheduler ports
	// are free before attempting the container start, but only when the
	// scheduler is publishing host ports. WSL2 commonly holds :2379 (etcd)
	// and the only reliable fix requires an elevated terminal.
	isWindowsHostPortMode := info.dockerNetwork == "" && runtime.GOOS == daprWindowsOS && isWSLAvailable()
	shouldWarnNonElevated := isWindowsHostPortMode && !isWindowsElevated()
	shouldManageWSL := isWindowsHostPortMode && isWindowsElevated()

	if shouldWarnNonElevated {
		if portErr := checkSchedulerPorts(osPort); portErr != nil {
			errorChan <- fmt.Errorf(
				"failed to start scheduler service: %v\n\n"+
					"A required port is already in use (often due to WSL).\n"+
					"To resolve this, re-run 'dapr init' in an elevated (Administrator)\n"+
					"terminal (e.g. right-click → \"Run as administrator\"). When running\n"+
					"elevated, the CLI will automatically stop and restart WSL and\n"+
					"Windows networking services as part of the installation process",
				portErr)
			return
		}
	}

	// On elevated Windows with host-port publishing and WSL2 installed, shut
	// down WSL2 and stop WinNAT so Docker can re-acquire the scheduler's port
	// bindings (especially etcd :2379) that WSL2 may be holding.
	// Skipped when using a Docker network (no host ports) or when WSL is not
	// present, to avoid unnecessary service disruption.
	winNATStopped := false
	if shouldManageWSL {
		print.InfoStatusEvent(os.Stdout, "Temporarily shutting down WSL to free ports for scheduler installation...")
		if wslErr := shutdownWSL(); wslErr != nil {
			print.WarningStatusEvent(os.Stdout, "Failed to shut down WSL: %v. Continuing...", wslErr)
		}
		print.InfoStatusEvent(os.Stdout, "Temporarily stopping Windows NAT service to free scheduler ports...")
		if stopErr := stopWinNAT(); stopErr != nil {
			print.WarningStatusEvent(os.Stdout, "Failed to stop Windows NAT service: %v. Continuing...", stopErr)
		} else {
			winNATStopped = true
		}
	}

	_, err = utils.RunCmdAndWait(runtimeCmd, args...)

	// Restore WinNAT and restart WSL regardless of whether the scheduler container started successfully.
	if info.dockerNetwork == "" && runtime.GOOS == daprWindowsOS && isWindowsElevated() && isWSLAvailable() {
		if winNATStopped {
			if startErr := startWinNAT(); startErr != nil {
				print.WarningStatusEvent(os.Stdout, "Failed to restart Windows NAT service: %v", startErr)
			}
		}
		print.InfoStatusEvent(os.Stdout, "Restarting WSL...")
		startWSLBackground()
	}

	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			errorChan <- parseContainerRuntimeError("scheduler service", err)
		} else {
			errorChan <- fmt.Errorf("%s %s failed with: %w", runtimeCmd, args, err)
		}
		return
	}
	errorChan <- nil
}

func generateCertsForMTLS(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if err := generateCertsForMTLSInternal(info); err != nil {
		errorChan <- err
	}
}

func generateCertsForMTLSInternal(info initInfo) error {
	if !info.enableMTLS {
		return nil
	}

	certsDir := GetDaprCertsPath(info.installDir)

	if err := os.MkdirAll(certsDir, 0o755); err != nil {
		return fmt.Errorf("error creating certs directory: %w", err)
	}

	caPath := path_filepath.Join(certsDir, trustAnchorsFile)
	issuerCertPath := path_filepath.Join(certsDir, issuerCertFile)
	issuerKeyPath := path_filepath.Join(certsDir, issuerKeyFile)

	if _, err := os.Stat(caPath); err == nil {
		if _, err := os.Stat(issuerCertPath); err == nil {
			if _, err := os.Stat(issuerKeyPath); err == nil {
				return nil
			}
		}
	}

	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("error generating root key for mTLS: %w", err)
	}

	certValidity := 365 * 24 * time.Hour
	certBundle, err := bundle.GenerateX509(bundle.OptionsX509{
		X509RootKey:   rootKey,
		TrustDomain:   defaultTrustDomain,
		OverrideCATTL: &certValidity,
	})
	if err != nil {
		return fmt.Errorf("error generating mTLS certificates: %w", err)
	}

	if err := os.WriteFile(caPath, certBundle.TrustAnchors, 0o600); err != nil {
		return fmt.Errorf("error writing CA certificate: %w", err)
	}
	if err := os.WriteFile(issuerCertPath, certBundle.IssChainPEM, 0o600); err != nil {
		return fmt.Errorf("error writing issuer certificate: %w", err)
	}
	if err := os.WriteFile(issuerKeyPath, certBundle.IssKeyPEM, 0o600); err != nil {
		return fmt.Errorf("error writing issuer key: %w", err)
	}
	return nil
}

func runSentryService(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if err := runSentryServiceInternal(info); err != nil {
		errorChan <- err
	}
}

func runSentryServiceInternal(info initInfo) error {
	if !info.enableMTLS || info.slimMode {
		return nil
	}

	runtimeCmd := utils.GetContainerRuntimeCmd(info.containerRuntime)
	sentryContainerName := utils.CreateContainerName(DaprSentryContainerName, info.dockerNetwork)

	exists, err := confirmContainerIsRunningOrExists(sentryContainerName, false, runtimeCmd)
	if err != nil {
		return err
	} else if exists {
		return fmt.Errorf("%s container exists or is running. %s", sentryContainerName, errInstallTemplate)
	}

	var image string

	imgInfo := daprImageInfo{
		ghcrImageName:      daprGhcrImageName,
		dockerHubImageName: daprDockerImageName,
		imageRegistryURL:   info.imageRegistryURL,
		imageRegistryName:  defaultImageRegistryName,
	}

	if isAirGapInit {
		dir := path_filepath.Join(info.fromDir, *info.bundleDet.ImageSubDir)
		image = info.bundleDet.getDaprImageName()
		err = loadContainer(dir, info.bundleDet.getDaprImageFileName(), info.containerRuntime)
		if err != nil {
			return err
		}
	} else {
		image, err = getDaprImageName(imgInfo, info)
		if err != nil {
			return err
		}
	}
	
	args := buildSentryContainerRunArgs(info, image)

	_, err = utils.RunCmdAndWait(runtimeCmd, args...)
	if err != nil {
		runError := isContainerRunError(err)
		if !runError {
			return parseContainerRuntimeError("sentry service", err)
		} else {
			return fmt.Errorf("%s %s failed with: %w", runtimeCmd, args, err)
		}
	}
	return nil
}

// checkSchedulerPorts verifies that all ports required by the scheduler
// service are available. grpcPort is the platform-specific gRPC port
// (50006 on Linux/Mac, 6060 on Windows).
func checkSchedulerPorts(grpcPort int) error {
	return checkPorts(grpcPort, schedulerEtcdPort, schedulerHealthPort, schedulerMetricPort)
}

// checkPorts returns an error for the first port in the list that is not
// available, including the port number in the message.
func checkPorts(ports ...int) error {
	for _, p := range ports {
		if err := utils.CheckIfPortAvailable(p); err != nil {
			return fmt.Errorf("port %d is not available: %w", p, err)
		}
	}
	return nil
}

func schedulerOverrideHostPort(info initInfo) bool {
	if info.runtimeVersion == "edge" || info.runtimeVersion == "dev" {
		return true
	}

	runV, err := semver.NewVersion(info.runtimeVersion)
	if err != nil {
		return true
	}

	v115rc5, _ := semver.NewVersion("1.15.0-rc.5")

	return runV.GreaterThan(v115rc5)
}

func schedulerEtcdClientListenAddress(info initInfo) bool {
	if info.runtimeVersion == "edge" || info.runtimeVersion == "dev" {
		return true
	}

	runV, err := semver.NewVersion(info.runtimeVersion)
	if err != nil {
		return true
	}

	v1160, _ := semver.NewVersion("1.16.0")

	return runV.GreaterThan(v1160)
}

func installDaprRuntime(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	err := installBinary(info.runtimeVersion, daprRuntimeFilePrefix, cli_ver.DaprGitHubRepo, info)
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

func installScheduler(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if !info.slimMode {
		return
	}

	hasScheduler, err := isSchedulerIncluded(info.runtimeVersion)
	if err != nil {
		errorChan <- err
		return
	}
	if !hasScheduler {
		return
	}

	err = installBinary(info.runtimeVersion, schedulerServiceFilePrefix, cli_ver.DaprGitHubRepo, info)
	if err != nil {
		errorChan <- err
	}
}

// installBinary installs the daprd, placement, or scheduler binaries and associated files inside the default dapr bin directory.
func installBinary(version, binaryFilePrefix, githubRepo string, info initInfo) error {
	var (
		err      error
		filepath string
	)

	dir := getDaprBinPath(info.installDir)
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

	// Make default components & config.
	componentsDir := GetDaprComponentsPath(info.installDir)
	configPath := GetDaprConfigPath(info.installDir)

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
	err = createDefaultConfiguration(zipkinHost, configPath, info.enableMTLS)
	if err != nil {
		errorChan <- fmt.Errorf("error creating default configuration file: %w", err)
		return
	}
}

func createSlimConfiguration(wg *sync.WaitGroup, errorChan chan<- error, info initInfo) {
	defer wg.Done()

	if !info.slimMode && !isAirGapInit {
		return
	}

	configPath := GetDaprConfigPath(info.installDir)
	// For --slim we pass empty string so that we do not configure zipkin.
	err := createDefaultConfiguration("", configPath, info.enableMTLS)
	if err != nil {
		errorChan <- fmt.Errorf("error creating default configuration file: %w", err)
		return
	}
}

func makeDefaultComponentsDir(installDir string) error {
	// Make default components directory.
	componentsDir := GetDaprComponentsPath(installDir)

	_, err := os.Stat(componentsDir)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(componentsDir, 0o755)
		if errDir != nil {
			return fmt.Errorf("error creating default components folder: %w", errDir)
		}
	}

	os.Chmod(componentsDir, 0o755)
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

		if strings.HasSuffix(fpath, binaryFilePrefix+".exe") {
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

		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode)) //nolint:gosec
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

	input, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	err = utils.CreateDirectory(destDir)
	if err != nil {
		return "", err
	}

	// #nosec G306
	if err = os.WriteFile(destFilePath, input, 0o644); err != nil {
		if runtime.GOOS != daprWindowsOS && strings.Contains(err.Error(), "permission denied") {
			err = errors.New(err.Error() + " - please run with sudo")
		}
		return "", err
	}

	if runtime.GOOS == daprWindowsOS {
		p := os.Getenv("PATH")

		if !strings.Contains(strings.ToLower(p), strings.ToLower(destDir)) {
			destDir = utils.SanitizeDir(destDir)
			pathCmd := "[System.Environment]::SetEnvironmentVariable('Path',[System.Environment]::GetEnvironmentVariable('Path','user') + '" + ";" + destDir + "', 'user')"
			_, err := utils.RunCmdAndWait("powershell", pathCmd)
			if err != nil {
				return "", err
			}
		}

		return destDir + "\\daprd.exe", nil
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
			Value: redisHost + ":6379",
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
			Value: redisHost + ":6379",
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

func createDefaultConfiguration(zipkinHost, filePath string, enableMTLS bool) error {
	defaultConfig := configuration{
		APIVersion: "dapr.io/v1alpha1",
		Kind:       "Configuration",
	}
	defaultConfig.Metadata.Name = "daprConfig"
	if zipkinHost != "" {
		defaultConfig.Spec.Tracing.SamplingRate = "1"
		defaultConfig.Spec.Tracing.Zipkin.EndpointAddress = fmt.Sprintf("http://%s:9411/api/v2/spans", zipkinHost) //nolint:nosprintfhostport
	}
	if enableMTLS {
		defaultConfig.Spec.MTLS.Enabled = true
	}
	b, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return err
	}

	if enableMTLS {
		if _, err := os.Stat(filePath); err == nil {
			return mergeMTLSIntoConfiguration(filePath)
		}
	}

	err = checkAndOverWriteFile(filePath, b)

	return err
}

func checkAndOverWriteFile(filePath string, b []byte) error {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// #nosec G306
		if err = os.WriteFile(filePath, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func prepareDaprInstallDir(daprBinDir string) error {
	err := os.MkdirAll(daprBinDir, 0o755)
	if err != nil {
		return err
	}

	err = os.Chmod(daprBinDir, 0o755)
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
			Proxy:                 http.ProxyFromEnvironment,
		},
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("version not found from url: %s", url)
	} else if resp.StatusCode != http.StatusOK {
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

/*
!
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

// getDaprImageName returns the resolved Dapr image name for online `dapr init`.
// It can either be resolved to the image-registry if given, otherwise GitHub container registry if
// selected or fallback to Docker Hub.
func getDaprImageName(imageInfo daprImageInfo, info initInfo) (string, error) {
	image, err := resolveImageURI(imageInfo)
	if err != nil {
		return "", err
	}

	image, err = getDaprImageWithTag(image, info.runtimeVersion, info.imageVariant)
	if err != nil {
		return "", err
	}

	// if default registry is GHCR and the image is not available in or cannot be pulled from GHCR
	// fallback to using dockerhub.
	if useGHCR(imageInfo, info.fromDir) && !tryPullImage(image, info.containerRuntime) {
		print.InfoStatusEvent(os.Stdout, "Image not found in Github container registry, pulling it from Docker Hub")
		image, err = getDaprImageWithTag(daprDockerImageName, info.runtimeVersion, info.imageVariant)
		if err != nil {
			return "", err
		}
	}
	return image, nil
}

func getDaprImageWithTag(name, version, imageVariant string) (string, error) {
	err := utils.ValidateImageVariant(imageVariant)
	if err != nil {
		return "", err
	}
	version = utils.GetVariantVersion(version, imageVariant)
	return fmt.Sprintf("%s:%s", name, version), nil
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
		if imageInfo.imageRegistryURL == ghcrURI || imageInfo.imageRegistryURL == dockerURI {
			return "", fmt.Errorf("flag --image-registry not set correctly. It cannot be %q or %q", ghcrURI, dockerURI)
		}
		return imageInfo.imageRegistryURL + "/" + imageInfo.ghcrImageName, nil
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
