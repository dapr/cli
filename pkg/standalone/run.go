// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"

	"github.com/Pallinder/sillyname-go"
	"github.com/dapr/dapr/pkg/components"
	modes "github.com/dapr/dapr/pkg/config/modes"
	"github.com/phayes/freeport"
	"gopkg.in/yaml.v2"
)

const sentryDefaultAddress = "localhost:50001"

// RunConfig represents the application configuration parameters.
type RunConfig struct {
	AppID              string `env:"APP_ID" arg:"app-id"`
	AppPort            int    `env:"APP_PORT" arg:"app-port"`
	HTTPPort           int    `env:"DAPR_HTTP_PORT" arg:"dapr-http-port"`
	GRPCPort           int    `env:"DAPR_GRPC_PORT" arg:"dapr-grpc-port"`
	ConfigFile         string `arg:"config"`
	Protocol           string `arg:"app-protocol"`
	Arguments          []string
	EnableProfiling    bool   `arg:"enable-profiling"`
	ProfilePort        int    `env:"DAPR_PROFILE_PORT" arg:"profile-port"`
	LogLevel           string `arg:"log-level"`
	MaxConcurrency     int    `arg:"app-max-concurrency"`
	PlacementHostAddr  string `env:"HOST_ADDRESS" arg:"placement-host-address"`
	ComponentsPath     string `arg:"components-path"`
	AppSSL             bool   `arg:"app-ssl"`
	MetricsPort        int    `env:"DAPR_METRICS_PORT" arg:"metrics-port"`
	MaxRequestBodySize int    `arg:"dapr-http-max-request-size"`
}

func (stat *DaprStat) newAppID() (*string, error) {
	for true {
		appID := strings.ReplaceAll(sillyname.GenerateStupidName(), " ", "-")
		if !stat.idExists(appID) {
			return &appID, nil
		}
	}
	return nil, nil
}

func (config *RunConfig) validateComponentPath() error {
	_, err := os.Stat(config.ComponentsPath)
	if err != nil {
		return err
	}
	componentsLoader := components.NewStandaloneComponents(modes.StandaloneConfig{ComponentsPath: config.ComponentsPath})
	_, err = componentsLoader.LoadComponents()
	if err != nil {
		return err
	}
	return nil
}

func (config *RunConfig) validatePlacementHostAddr() error {
	placementHostAddr := config.PlacementHostAddr
	if indx := strings.Index(placementHostAddr, ":"); indx == -1 {
		if runtime.GOOS == daprWindowsOS {
			placementHostAddr = fmt.Sprintf("%s:6050", placementHostAddr)
		} else {
			placementHostAddr = fmt.Sprintf("%s:50005", placementHostAddr)
		}
		config.PlacementHostAddr = placementHostAddr
	}
	return nil
}

func (config *RunConfig) validatePort(context string, portPtr *int, stat *DaprStat) error {
	if *portPtr < 0 {
		port, err := freeport.GetFreePort()
		if err != nil {
			return err
		}
		*portPtr = port
		return nil
	}

	if stat.portExists(*portPtr) {
		return fmt.Errorf("Invalid configuration for %s. Port %v is not available", context, *portPtr)
	}
	return nil
}

func (config *RunConfig) validate() error {
	stat, err := newDaprStat()
	if err != nil {
		return err
	}

	if config.AppID == "" {
		appId, err := stat.newAppID()
		if err != nil {
			return err
		}
		config.AppID = *appId
	}

	err = config.validateComponentPath()
	if err != nil {
		return err
	}

	err = config.validatePort("AppPort", &config.AppPort, stat)
	if err != nil {
		return err
	}

	err = config.validatePort("HTTPPort", &config.HTTPPort, stat)
	if err != nil {
		return err
	}

	err = config.validatePort("GRPCPort", &config.GRPCPort, stat)
	if err != nil {
		return err
	}

	err = config.validatePort("MetricsPort", &config.MetricsPort, stat)
	if err != nil {
		return err
	}

	if config.EnableProfiling {
		err = config.validatePort("ProfilePort", &config.ProfilePort, stat)
		if err != nil {
			return err
		}
	}

	if config.MaxConcurrency < 1 {
		config.MaxConcurrency = -1
	}
	if config.MaxRequestBodySize < 0 {
		config.MaxRequestBodySize = -1
	}

	err = config.validatePlacementHostAddr()
	if err != nil {
		return err
	}
	return nil
}

type DaprStat struct {
	ExistingIDs   map[string]bool
	ExistingPorts map[int]bool
}

func (stat *DaprStat) idExists(id string) bool {
	_, ok := stat.ExistingIDs[id]
	return ok
}

func (stat *DaprStat) portExists(port int) bool {
	_, ok := stat.ExistingPorts[port]
	if !ok {
		available := stat.portAvailable(port)
		if available {
			stat.ExistingPorts[port] = true
			ok = false
		}
	}
	return ok
}

func (stat *DaprStat) portAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		return false
	}

	listener.Close()
	return true
}

func newDaprStat() (*DaprStat, error) {
	stat := DaprStat{}
	stat.ExistingIDs = make(map[string]bool)
	stat.ExistingPorts = make(map[int]bool)
	dapr, err := List()
	if err != nil {
		return nil, err
	}
	for _, instance := range dapr {
		stat.ExistingIDs[instance.AppID] = true
		stat.ExistingPorts[instance.AppPort] = true
		stat.ExistingPorts[instance.HTTPPort] = true
		stat.ExistingPorts[instance.GRPCPort] = true
	}
	return &stat, nil
}
func (config *RunConfig) getArgs() []string {
	args := []string{}
	schema := reflect.ValueOf(*config)
	for i := 0; i < schema.NumField(); i++ {
		valueField := schema.Field(i).Interface()
		typeField := schema.Type().Field(i)
		key := typeField.Tag.Get("arg")
		if len(key) == 0 {
			continue
		}
		key = "--" + key

		switch valueField.(type) {
		case bool:
			if valueField == true {
				args = append(args, key)
			}
		default:
			value := fmt.Sprintf("%v", reflect.ValueOf(valueField))
			if len(value) != 0 {
				args = append(args, key, value)
			}
		}
	}
	if config.ConfigFile != "" {
		sentryAddress := mtlsEndpoint(config.ConfigFile)
		if sentryAddress != "" {
			// mTLS is enabled locally, set it up
			args = append(args, "--enable-mtls", "--sentry-address", sentryAddress)
		}
	}

	return args
}

func (config *RunConfig) getEnv() []string {
	env := []string{}
	schema := reflect.ValueOf(*config)
	for i := 0; i < schema.NumField(); i++ {
		valueField := schema.Field(i)
		typeField := schema.Type().Field(i)
		key := typeField.Tag.Get("env")
		if len(key) == 0 {
			continue
		}
		value := reflect.ValueOf(valueField.Interface())

		env = append(env, fmt.Sprintf("%s=%v", key, value))
	}
	return env
}

// RunOutput represents the run output.
type RunOutput struct {
	DaprCMD      *exec.Cmd
	DaprHTTPPort int
	DaprGRPCPort int
	AppID        string
	AppCMD       *exec.Cmd
}

func getDaprCommand(config *RunConfig) (*exec.Cmd, error) {
	daprCMD := binaryFilePath(defaultDaprBinPath(), "daprd")
	args := config.getArgs()
	cmd := exec.Command(daprCMD, args...)
	return cmd, nil
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

func getAppCommand(config *RunConfig) *exec.Cmd {
	argCount := len(config.Arguments)

	if argCount == 0 {
		return nil
	}
	command := config.Arguments[0]

	args := []string{}
	if argCount > 1 {
		args = config.Arguments[1:]
	}

	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, config.getEnv()...)

	return cmd
}

func Run(config *RunConfig) (*RunOutput, error) {
	err := config.validate()
	if err != nil {
		return nil, err
	}

	daprCMD, err := getDaprCommand(config)
	if err != nil {
		return nil, err
	}

	var appCMD *exec.Cmd = getAppCommand(config)
	return &RunOutput{
		DaprCMD:      daprCMD,
		AppCMD:       appCMD,
		AppID:        config.AppID,
		DaprHTTPPort: config.HTTPPort,
		DaprGRPCPort: config.GRPCPort,
	}, nil
}
