// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/Pallinder/sillyname-go"
	"github.com/phayes/freeport"

	"github.com/dapr/dapr/pkg/components"
	modes "github.com/dapr/dapr/pkg/config/modes"
)

const sentryDefaultAddress = "localhost:50001"

// RunConfig represents the application configuration parameters.
type RunConfig struct {
	AppID           string
	AppPort         int
	HTTPPort        int
	GRPCPort        int
	ConfigFile      string
	Protocol        string
	Arguments       []string
	EnableProfiling bool
	ProfilePort     int
	LogLevel        string
	MaxConcurrency  int
	PlacementHost   string
	ComponentsPath  string
}

// RunOutput represents the run output.
type RunOutput struct {
	DaprCMD      *exec.Cmd
	DaprHTTPPort int
	DaprGRPCPort int
	AppID        string
	AppCMD       *exec.Cmd
}

func getDaprCommand(appID string, daprHTTPPort int, daprGRPCPort int, appPort int, configFile, protocol string, enableProfiling bool, profilePort int, logLevel string, maxConcurrency int, placementHost string, componentsPath string) (*exec.Cmd, int, int, int, error) {
	if daprHTTPPort < 0 {
		port, err := freeport.GetFreePort()
		if err != nil {
			return nil, -1, -1, -1, err
		}

		daprHTTPPort = port
	}

	if daprGRPCPort < 0 {
		grpcPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, -1, -1, -1, err
		}

		daprGRPCPort = grpcPort
	}

	if maxConcurrency < 1 {
		maxConcurrency = -1
	}

	daprCMD := binaryFilePath(defaultDaprDirPath(), "daprd")
	metricsPort, err := freeport.GetFreePort()
	if err != nil {
		return nil, -1, -1, -1, err
	}

	args := []string{"--app-id", appID, "--dapr-http-port", fmt.Sprintf("%v", daprHTTPPort), "--dapr-grpc-port", fmt.Sprintf("%v", daprGRPCPort), "--log-level", logLevel, "--max-concurrency", fmt.Sprintf("%v", maxConcurrency), "--protocol", protocol, "--metrics-port", fmt.Sprintf("%v", metricsPort), "--components-path", componentsPath}
	if appPort > -1 {
		args = append(args, "--app-port", fmt.Sprintf("%v", appPort))
	}

	args = append(args, "--placement-address")

	if runtime.GOOS == daprWindowsOS {
		args = append(args, fmt.Sprintf("%s:6050", placementHost))
	} else {
		args = append(args, fmt.Sprintf("%s:50005", placementHost))
	}

	if configFile != "" {
		args = append(args, "--config", configFile)
		sentryAddress := mtlsEndpoint(configFile)
		if sentryAddress != "" {
			// mTLS is enabled locally, set it up
			args = append(args, "--enable-mtls", "--sentry-address", sentryAddress)
		}
	}

	if enableProfiling {
		if profilePort == -1 {
			pp, err := freeport.GetFreePort()
			if err != nil {
				return nil, -1, -1, -1, err
			}
			profilePort = pp
		}

		args = append(
			args,
			"--enable-profiling", "true",
			"--profile-port", fmt.Sprintf("%v", profilePort))
	}

	cmd := exec.Command(daprCMD, args...)
	return cmd, daprHTTPPort, daprGRPCPort, metricsPort, nil
}

func mtlsEndpoint(configFile string) string {
	if configFile == "" {
		return ""
	}

	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return ""
	}

	var config mtlsConfig
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return ""
	}

	if config.Spec.MTLS.Enabled {
		return sentryDefaultAddress
	}
	return ""
}

func getAppCommand(httpPort, grpcPort, metricsPort int, command string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(
		cmd.Env,
		fmt.Sprintf("DAPR_HTTP_PORT=%v", httpPort),
		fmt.Sprintf("DAPR_GRPC_PORT=%v", grpcPort),
		fmt.Sprintf("DAPR_METRICS_PORT=%v", metricsPort))

	return cmd, nil
}

func Run(config *RunConfig) (*RunOutput, error) {
	appID := config.AppID
	if appID == "" {
		appID = strings.Replace(sillyname.GenerateStupidName(), " ", "-", -1)
	}

	_, err := os.Stat(config.ComponentsPath)
	if err != nil {
		return nil, err
	}

	dapr, err := List()
	if err != nil {
		return nil, err
	}

	for _, a := range dapr {
		if appID == a.AppID {
			return nil, fmt.Errorf("dapr with ID %s is already running", appID)
		}
	}

	componentsLoader := components.NewStandaloneComponents(modes.StandaloneConfig{ComponentsPath: config.ComponentsPath})
	_, err = componentsLoader.LoadComponents()
	if err != nil {
		return nil, err
	}

	daprCMD, daprHTTPPort, daprGRPCPort, metricsPort, err := getDaprCommand(appID, config.HTTPPort, config.GRPCPort, config.AppPort, config.ConfigFile, config.Protocol, config.EnableProfiling, config.ProfilePort, config.LogLevel, config.MaxConcurrency, config.PlacementHost, config.ComponentsPath)

	if err != nil {
		return nil, err
	}

	for _, a := range dapr {
		if daprHTTPPort == a.HTTPPort {
			return nil, fmt.Errorf("there's already a Dapr instance running with http port %v", daprHTTPPort)
		} else if daprGRPCPort == a.GRPCPort {
			return nil, fmt.Errorf("there's already a Dapr instance running with gRPC port %v", daprGRPCPort)
		}
	}

	argCount := len(config.Arguments)
	runArgs := []string{}
	var appCMD *exec.Cmd

	if argCount > 0 {
		cmd := config.Arguments[0]
		if len(config.Arguments) > 1 {
			runArgs = config.Arguments[1:]
		}

		appCMD, err = getAppCommand(daprHTTPPort, daprGRPCPort, metricsPort, cmd, runArgs)
		if err != nil {
			return nil, err
		}
	}

	return &RunOutput{
		DaprCMD:      daprCMD,
		AppCMD:       appCMD,
		AppID:        appID,
		DaprHTTPPort: daprHTTPPort,
		DaprGRPCPort: daprGRPCPort,
	}, nil
}
