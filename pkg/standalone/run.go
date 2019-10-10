// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/Pallinder/sillyname-go"
	"github.com/phayes/freeport"
)

const (
	componentsDirName           = "components"
	redisMessageBusYamlFileName = "redis_messagebus.yaml"
	redisStateStoreYamlFileName = "redis.yaml"
)

// RunConfig to represent application configuration parameters
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
}

// RunOutput to represent the run output
type RunOutput struct {
	DaprCMD      *exec.Cmd
	DaprHTTPPort int
	DaprGRPCPort int
	AppID        string
	AppCMD       *exec.Cmd
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

func getDaprCommand(appID string, daprHTTPPort int, daprGRPCPort int, appPort int, configFile, protocol string, enableProfiling bool, profilePort int, logLevel string, maxConcurrency int) (*exec.Cmd, int, int, error) {
	if daprHTTPPort < 0 {
		port, err := freeport.GetFreePort()
		if err != nil {
			return nil, -1, -1, err
		}

		daprHTTPPort = port
	}

	if daprGRPCPort < 0 {
		grpcPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, -1, -1, err
		}

		daprGRPCPort = grpcPort
	}

	if maxConcurrency < 1 {
		maxConcurrency = -1
	}

	daprCMD := "daprd"
	if runtime.GOOS == "windows" {
		daprCMD = fmt.Sprintf("%s.exe", daprCMD)
	}

	args := []string{"--dapr-id", appID, "--dapr-http-port", fmt.Sprintf("%v", daprHTTPPort), "--dapr-grpc-port", fmt.Sprintf("%v", daprGRPCPort), "--log-level", logLevel, "--max-concurrency", fmt.Sprintf("%v", maxConcurrency), "--protocol", protocol}
	if appPort > -1 {
		args = append(args, "--app-port")
		args = append(args, fmt.Sprintf("%v", appPort))
	}

	args = append(args, "--placement-address")

	if runtime.GOOS == "windows" {
		args = append(args, "localhost:6050")
	} else {
		args = append(args, "localhost:50005")
	}

	if configFile != "" {
		args = append(args, "--config")
		args = append(args, configFile)
	}

	if enableProfiling {
		if profilePort == -1 {
			pp, err := freeport.GetFreePort()
			if err != nil {
				return nil, -1, -1, err
			}
			profilePort = pp
		}

		args = append(args, "--enable-profiling")
		args = append(args, "true")
		args = append(args, "--profile-port")
		args = append(args, fmt.Sprintf("%v", profilePort))
	}

	cmd := exec.Command(daprCMD, args...)
	return cmd, daprHTTPPort, daprGRPCPort, nil
}

func getAppCommand(httpPort, grpcPort int, command string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("DAPR_HTTP_PORT=%v", httpPort))
	cmd.Env = append(cmd.Env, fmt.Sprintf("DAPR_GRPC_PORT=%v", grpcPort))

	return cmd, nil
}

func dirOrFileExists(dirOrFilePath string) bool {
	_, err := os.Stat(dirOrFilePath)
	return !os.IsNotExist(err)
}

func absoluteComponentsDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return path.Join(wd, componentsDirName), nil
}

func createRedisStateStore() error {
	redisStore := component{
		APIVersion: "dapr.io/v1alpha1",
		Kind:       "Component",
	}

	redisStore.Metadata.Name = "statestore"
	redisStore.Spec.Type = "state.redis"
	redisStore.Spec.Metadata = []componentMetadataItem{}
	redisStore.Spec.Metadata = append(redisStore.Spec.Metadata, componentMetadataItem{
		Name:  "redisHost",
		Value: "localhost:6379",
	})
	redisStore.Spec.Metadata = append(redisStore.Spec.Metadata, componentMetadataItem{
		Name:  "redisPassword",
		Value: "",
	})

	b, err := yaml.Marshal(&redisStore)
	if err != nil {
		return err
	}

	componentsDir, err := absoluteComponentsDir()
	if err != nil {
		return err
	}

	os.Mkdir(componentsDir, 0777)
	err = ioutil.WriteFile(path.Join(componentsDir, redisStateStoreYamlFileName), b, 0644)
	if err != nil {
		return err
	}

	return nil
}

func createRedisPubSub() error {
	redisMessageBus := component{
		APIVersion: "dapr.io/v1alpha1",
		Kind:       "Component",
	}

	redisMessageBus.Metadata.Name = "messagebus"
	redisMessageBus.Spec.Type = "pubsub.redis"
	redisMessageBus.Spec.Metadata = []componentMetadataItem{}
	redisMessageBus.Spec.Metadata = append(redisMessageBus.Spec.Metadata, componentMetadataItem{
		Name:  "redisHost",
		Value: "localhost:6379",
	})
	redisMessageBus.Spec.Metadata = append(redisMessageBus.Spec.Metadata, componentMetadataItem{
		Name:  "redisPassword",
		Value: "",
	})

	b, err := yaml.Marshal(&redisMessageBus)
	if err != nil {
		return err
	}

	componentsDir, err := absoluteComponentsDir()
	if err != nil {
		return err
	}

	os.Mkdir(componentsDir, 0777)
	err = ioutil.WriteFile(path.Join(componentsDir, redisMessageBusYamlFileName), b, 0644)
	if err != nil {
		return err
	}

	return nil
}

func Run(config *RunConfig) (*RunOutput, error) {
	appID := config.AppID
	if appID == "" {
		appID = strings.Replace(sillyname.GenerateStupidName(), " ", "-", -1)
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

	componentsDir, err := absoluteComponentsDir()
	if err != nil {
		return nil, err
	}

	if !dirOrFileExists(path.Join(componentsDir, redisStateStoreYamlFileName)) {
		err = createRedisStateStore()
		if err != nil {
			return nil, err
		}
	}

	if !dirOrFileExists(path.Join(componentsDir, redisMessageBusYamlFileName)) {
		err = createRedisPubSub()
		if err != nil {
			return nil, err
		}
	}

	daprCMD, daprHTTPPort, daprGRPCPort, err := getDaprCommand(appID, config.HTTPPort, config.GRPCPort, config.AppPort, config.ConfigFile, config.Protocol, config.EnableProfiling, config.ProfilePort, config.LogLevel, config.MaxConcurrency)
	if err != nil {
		return nil, err
	}

	for _, a := range dapr {
		if daprHTTPPort == a.HTTPPort {
			return nil, fmt.Errorf("there's already a dapr instance running with http port %v", daprHTTPPort)
		} else if daprGRPCPort == a.GRPCPort {
			return nil, fmt.Errorf("there's already a dapr instance running with gRPC port %v", daprGRPCPort)
		}
	}

	runArgs := []string{}
	argCount := len(config.Arguments)

	if argCount == 0 {
		return nil, errors.New("No app entrypoint given")
	}

	cmd := config.Arguments[0]
	if len(config.Arguments) > 1 {
		runArgs = config.Arguments[1:]
	}

	appCMD, err := getAppCommand(daprHTTPPort, daprGRPCPort, cmd, runArgs)
	if err != nil {
		return nil, err
	}

	return &RunOutput{
		DaprCMD:      daprCMD,
		AppCMD:       appCMD,
		AppID:        appID,
		DaprHTTPPort: daprHTTPPort,
		DaprGRPCPort: daprGRPCPort,
	}, nil
}
