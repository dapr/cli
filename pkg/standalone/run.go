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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"

	"github.com/Pallinder/sillyname-go"
	"github.com/phayes/freeport"
	"gopkg.in/yaml.v2"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/dapr/pkg/components"
	modes "github.com/dapr/dapr/pkg/config/modes"
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
	ProfilePort        int    `arg:"profile-port"`
	LogLevel           string `arg:"log-level"`
	MaxConcurrency     int    `arg:"app-max-concurrency"`
	PlacementHostAddr  string `arg:"placement-host-address"`
	ComponentsPath     string `arg:"components-path"`
	SpecResourcesPath  string `arg:"spec-resources-path"`
	AppSSL             bool   `arg:"app-ssl"`
	MetricsPort        int    `env:"DAPR_METRICS_PORT" arg:"metrics-port"`
	MaxRequestBodySize int    `arg:"dapr-http-max-request-size"`
	HTTPReadBufferSize int    `arg:"dapr-http-read-buffer-size"`
	UnixDomainSocket   string `arg:"unix-domain-socket"`
	EnableAPILogging   bool   `arg:"enable-api-logging"`
}

func (meta *DaprMeta) newAppID() string {
	for {
		appID := strings.ReplaceAll(sillyname.GenerateStupidName(), " ", "-")
		if !meta.idExists(appID) {
			return appID
		}
	}
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

func (config *RunConfig) validateSpecResourcesPath() error {
	_, err := os.Stat(config.SpecResourcesPath)
	if err != nil {
		return err
	}
	componentsLoader := components.NewStandaloneComponents(modes.StandaloneConfig{ComponentsPath: config.SpecResourcesPath})
	_, err = componentsLoader.LoadComponents()
	if err != nil {
		return err
	}
	return nil
}

func (config *RunConfig) validatePlacementHostAddr() error {
	placementHostAddr := config.PlacementHostAddr
	if len(placementHostAddr) == 0 {
		placementHostAddr = "localhost"
	}
	if indx := strings.Index(placementHostAddr, ":"); indx == -1 {
		if runtime.GOOS == daprWindowsOS {
			placementHostAddr = fmt.Sprintf("%s:6050", placementHostAddr)
		} else {
			placementHostAddr = fmt.Sprintf("%s:50005", placementHostAddr)
		}
	}
	config.PlacementHostAddr = placementHostAddr
	return nil
}

func (config *RunConfig) validatePort(portName string, portPtr *int, meta *DaprMeta) error {
	if *portPtr <= 0 {
		port, err := freeport.GetFreePort()
		if err != nil {
			return err
		}
		*portPtr = port
		return nil
	}

	if meta.portExists(*portPtr) {
		return fmt.Errorf("invalid configuration for %s. Port %v is not available", portName, *portPtr)
	}
	return nil
}

func (config *RunConfig) validate() error {
	meta, err := newDaprMeta()
	if err != nil {
		return err
	}

	if config.AppID == "" {
		config.AppID = meta.newAppID()
	}

	err = config.validateSpecResourcesPath()
	if err != nil {
		return err
	}

	err = config.validateComponentPath()
	if err != nil {
		return err
	}

	if config.AppPort < 0 {
		config.AppPort = 0
	}

	err = config.validatePort("HTTPPort", &config.HTTPPort, meta)
	if err != nil {
		return err
	}

	err = config.validatePort("GRPCPort", &config.GRPCPort, meta)
	if err != nil {
		return err
	}

	err = config.validatePort("MetricsPort", &config.MetricsPort, meta)
	if err != nil {
		return err
	}

	if config.EnableProfiling {
		err = config.validatePort("ProfilePort", &config.ProfilePort, meta)
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

	if config.HTTPReadBufferSize < 0 {
		config.HTTPReadBufferSize = -1
	}

	err = config.validatePlacementHostAddr()
	if err != nil {
		return err
	}
	return nil
}

type DaprMeta struct {
	ExistingIDs   map[string]bool
	ExistingPorts map[int]bool
}

func (meta *DaprMeta) idExists(id string) bool {
	_, ok := meta.ExistingIDs[id]
	return ok
}

func (meta *DaprMeta) portExists(port int) bool {
	if port <= 0 {
		return false
	}
	//nolint
	_, ok := meta.ExistingPorts[port]
	if ok {
		return true
	}

	// try to listen on the port.
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		return true
	}
	listener.Close()

	meta.ExistingPorts[port] = true
	return false
}

func newDaprMeta() (*DaprMeta, error) {
	meta := DaprMeta{}
	meta.ExistingIDs = make(map[string]bool)
	meta.ExistingPorts = make(map[int]bool)
	dapr, err := List()
	if err != nil {
		return nil, err
	}
	for _, instance := range dapr {
		meta.ExistingIDs[instance.AppID] = true
		meta.ExistingPorts[instance.AppPort] = true
		meta.ExistingPorts[instance.HTTPPort] = true
		meta.ExistingPorts[instance.GRPCPort] = true
	}
	return &meta, nil
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
			// mTLS is enabled locally, set it up.
			args = append(args, "--enable-mtls", "--sentry-address", sentryAddress)
		}
	}

	if print.IsJSONLogEnabled() {
		args = append(args, "--log-as-json")
	}

	return args
}

func (config *RunConfig) getEnv() []string {
	env := []string{}
	schema := reflect.ValueOf(*config)
	for i := 0; i < schema.NumField(); i++ {
		valueField := schema.Field(i).Interface()
		typeField := schema.Type().Field(i)
		key := typeField.Tag.Get("env")
		if len(key) == 0 {
			continue
		}
		if value, ok := valueField.(int); ok && value <= 0 {
			// ignore unset numeric variables.
			continue
		}

		value := fmt.Sprintf("%v", reflect.ValueOf(valueField))
		env = append(env, fmt.Sprintf("%s=%v", key, value))
	}
	return env
}

// RunOutput represents the run output.
type RunOutput struct {
	DaprCMD      *exec.Cmd
	DaprErr      error
	DaprHTTPPort int
	DaprGRPCPort int
	AppID        string
	AppCMD       *exec.Cmd
	AppErr       error
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
	//nolint
	err := config.validate()
	if err != nil {
		return nil, err
	}

	daprCMD, err := getDaprCommand(config)
	if err != nil {
		return nil, err
	}

	//nolint
	var appCMD *exec.Cmd = getAppCommand(config)
	return &RunOutput{
		DaprCMD:      daprCMD,
		DaprErr:      nil,
		AppCMD:       appCMD,
		AppErr:       nil,
		AppID:        config.AppID,
		DaprHTTPPort: config.HTTPPort,
		DaprGRPCPort: config.GRPCPort,
	}, nil
}
