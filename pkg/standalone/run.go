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
	"net"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/Pallinder/sillyname-go"
	"github.com/phayes/freeport"
	"gopkg.in/yaml.v2"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/dapr/pkg/components"
)

type LogDestType string

const (
	Console             LogDestType = "console"
	File                LogDestType = "file"
	FileAndConsole      LogDestType = "fileAndConsole"
	DefaultDaprdLogDest             = File
	DefaultAppLogDest               = FileAndConsole

	sentryDefaultAddress = "localhost:50001"
	defaultStructTagKey  = "default"
)

// RunConfig represents the application configuration parameters.
type RunConfig struct {
	SharedRunConfig   `yaml:",inline"`
	AppID             string   `env:"APP_ID" arg:"app-id" yaml:"appID"`
	AppChannelAddress string   `env:"APP_CHANNEL_ADDRESS" arg:"app-channel-address" ifneq:"127.0.0.1" yaml:"appChannelAddress"`
	AppPort           int      `env:"APP_PORT" arg:"app-port" yaml:"appPort" default:"-1"`
	HTTPPort          int      `env:"DAPR_HTTP_PORT" arg:"dapr-http-port" yaml:"daprHTTPPort" default:"-1"`
	GRPCPort          int      `env:"DAPR_GRPC_PORT" arg:"dapr-grpc-port" yaml:"daprGRPCPort" default:"-1"`
	ProfilePort       int      `arg:"profile-port" yaml:"profilePort" default:"-1"`
	Command           []string `yaml:"command"`
	MetricsPort       int      `env:"DAPR_METRICS_PORT" arg:"metrics-port" yaml:"metricsPort" default:"-1"`
	UnixDomainSocket  string   `arg:"unix-domain-socket" yaml:"unixDomainSocket"`
	InternalGRPCPort  int      `arg:"dapr-internal-grpc-port" yaml:"daprInternalGRPCPort" default:"-1"`
}

// SharedRunConfig represents the application configuration parameters, which can be shared across many apps.
type SharedRunConfig struct {
	ConfigFile          string            `arg:"config" yaml:"configFilePath"`
	AppProtocol         string            `arg:"app-protocol" yaml:"appProtocol" default:"http"`
	APIListenAddresses  string            `arg:"dapr-listen-addresses" yaml:"apiListenAddresses"`
	EnableProfiling     bool              `arg:"enable-profiling" yaml:"enableProfiling"`
	LogLevel            string            `arg:"log-level" yaml:"logLevel"`
	MaxConcurrency      int               `arg:"app-max-concurrency" yaml:"appMaxConcurrency" default:"-1"`
	PlacementHostAddr   string            `arg:"placement-host-address" yaml:"placementHostAddress"`
	ComponentsPath      string            `arg:"components-path"` // Deprecated in run template file: use ResourcesPaths instead.
	ResourcesPath       string            `yaml:"resourcesPath"`  // Deprecated in run template file: use ResourcesPaths instead.
	ResourcesPaths      []string          `arg:"resources-path" yaml:"resourcesPaths"`
	AppSSL              bool              `arg:"app-ssl" yaml:"appSSL"`
	MaxRequestBodySize  int               `arg:"dapr-http-max-request-size" yaml:"daprHTTPMaxRequestSize" default:"-1"`
	HTTPReadBufferSize  int               `arg:"dapr-http-read-buffer-size" yaml:"daprHTTPReadBufferSize" default:"-1"`
	EnableAppHealth     bool              `arg:"enable-app-health-check" yaml:"enableAppHealthCheck"`
	AppHealthPath       string            `arg:"app-health-check-path" yaml:"appHealthCheckPath"`
	AppHealthInterval   int               `arg:"app-health-probe-interval" ifneq:"0" yaml:"appHealthProbeInterval"`
	AppHealthTimeout    int               `arg:"app-health-probe-timeout" ifneq:"0" yaml:"appHealthProbeTimeout"`
	AppHealthThreshold  int               `arg:"app-health-threshold" ifneq:"0" yaml:"appHealthThreshold"`
	EnableAPILogging    bool              `arg:"enable-api-logging" yaml:"enableApiLogging"`
	DaprdInstallPath    string            `yaml:"runtimePath"`
	Env                 map[string]string `yaml:"env"`
	DaprdLogDestination LogDestType       `yaml:"daprdLogDestination"`
	AppLogDestination   LogDestType       `yaml:"appLogDestination"`
}

func (meta *DaprMeta) newAppID() string {
	for {
		appID := strings.ToLower(strings.ReplaceAll(sillyname.GenerateStupidName(), " ", "-"))
		if !meta.idExists(appID) {
			return appID
		}
	}
}

func (config *RunConfig) validateResourcesPaths() error {
	dirPath := config.ResourcesPaths
	if len(dirPath) == 0 {
		dirPath = []string{config.ComponentsPath}
	}
	for _, path := range dirPath {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("error validating resources path %q : %w", dirPath, err)
		}
	}
	componentsLoader := components.NewLocalComponents(dirPath...)
	_, err := componentsLoader.LoadComponents()
	if err != nil {
		return fmt.Errorf("error validating components in resources path %q : %w", dirPath, err)
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

func (config *RunConfig) Validate() error {
	meta, err := newDaprMeta()
	if err != nil {
		return err
	}

	if config.AppID == "" {
		config.AppID = meta.newAppID()
	}

	err = config.validateResourcesPaths()
	if err != nil {
		return err
	}

	if len(config.ResourcesPaths) > 0 {
		config.ComponentsPath = ""
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

	err = config.validatePort("InternalGRPCPort", &config.InternalGRPCPort, meta)
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
	args = getArgsFromSchema(schema, args)

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

// Recursive function to get all the args from the config struct.
// This is needed because the config struct has embedded struct.
func getArgsFromSchema(schema reflect.Value, args []string) []string {
	for i := 0; i < schema.NumField(); i++ {
		valueField := schema.Field(i).Interface()
		typeField := schema.Type().Field(i)
		key := typeField.Tag.Get("arg")
		if typeField.Type.Kind() == reflect.Struct {
			args = getArgsFromSchema(schema.Field(i), args)
			continue
		}
		if len(key) == 0 {
			continue
		}
		key = "--" + key

		ifneq, hasIfneq := typeField.Tag.Lookup("ifneq")

		switch vType := valueField.(type) {
		case bool:
			if valueField == true {
				args = append(args, key)
			}
		case []string:
			if len(vType) > 0 {
				for _, val := range vType {
					args = append(args, key, val)
				}
			}
		default:
			value := fmt.Sprintf("%v", reflect.ValueOf(valueField))
			if len(value) != 0 && (!hasIfneq || value != ifneq) {
				args = append(args, key, value)
			}
		}
	}
	return args
}

func (config *RunConfig) SetDefaultFromSchema() {
	schema := reflect.ValueOf(*config)
	config.setDefaultFromSchemaRecursive(schema)
}

func (config *RunConfig) setDefaultFromSchemaRecursive(schema reflect.Value) {
	for i := 0; i < schema.NumField(); i++ {
		valueField := schema.Field(i)
		typeField := schema.Type().Field(i)
		if typeField.Type.Kind() == reflect.Struct {
			config.setDefaultFromSchemaRecursive(valueField)
			continue
		}
		if valueField.IsZero() && len(typeField.Tag.Get(defaultStructTagKey)) != 0 {
			switch valueField.Kind() {
			case reflect.Int:
				if val, err := strconv.ParseInt(typeField.Tag.Get(defaultStructTagKey), 10, 64); err == nil {
					reflect.ValueOf(config).Elem().FieldByName(typeField.Name).Set(reflect.ValueOf(int(val)).Convert(valueField.Type()))
				}
			case reflect.String:
				val := typeField.Tag.Get(defaultStructTagKey)
				reflect.ValueOf(config).Elem().FieldByName(typeField.Name).Set(reflect.ValueOf(val).Convert(valueField.Type()))
			}
		}
	}
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
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%v", k, v))
	}
	return env
}

func GetDaprCommand(config *RunConfig) (*exec.Cmd, error) {
	daprCMD, err := lookupBinaryFilePath(config.DaprdInstallPath, "daprd")
	if err != nil {
		return nil, err
	}

	args := config.getArgs()
	cmd := exec.Command(daprCMD, args...)
	return cmd, nil
}

func mtlsEndpoint(configFile string) string {
	if configFile == "" {
		return ""
	}

	b, err := os.ReadFile(configFile)
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

func GetAppCommand(config *RunConfig) *exec.Cmd {
	argCount := len(config.Command)

	if argCount == 0 {
		return nil
	}
	command := config.Command[0]

	args := []string{}
	if argCount > 1 {
		args = config.Command[1:]
	}

	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, config.getEnv()...)

	return cmd
}

func (l LogDestType) String() string {
	return string(l)
}

func (l LogDestType) IsValid() error {
	switch l {
	case Console, File, FileAndConsole:
		return nil
	}
	return fmt.Errorf("invalid log destination type: %s", l)
}
