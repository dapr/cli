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
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/Pallinder/sillyname-go"
	"github.com/phayes/freeport"
	"gopkg.in/yaml.v2"

	"github.com/dapr/cli/pkg/print"
	localloader "github.com/dapr/dapr/pkg/components/loader"
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
	AppID             string   `env:"APP_ID" arg:"app-id" annotation:"dapr.io/app-id" yaml:"appID"`
	AppChannelAddress string   `env:"APP_CHANNEL_ADDRESS" arg:"app-channel-address" ifneq:"127.0.0.1" yaml:"appChannelAddress"`
	AppPort           int      `env:"APP_PORT" arg:"app-port" annotation:"dapr.io/app-port" yaml:"appPort" default:"-1"`
	HTTPPort          int      `env:"DAPR_HTTP_PORT" arg:"dapr-http-port" yaml:"daprHTTPPort" default:"-1"`
	GRPCPort          int      `env:"DAPR_GRPC_PORT" arg:"dapr-grpc-port" yaml:"daprGRPCPort" default:"-1"`
	ProfilePort       int      `arg:"profile-port" yaml:"profilePort" default:"-1"`
	Command           []string `yaml:"command"`
	MetricsPort       int      `env:"DAPR_METRICS_PORT" arg:"metrics-port" annotation:"dapr.io/metrics-port" yaml:"metricsPort" default:"-1"`
	UnixDomainSocket  string   `arg:"unix-domain-socket" annotation:"dapr.io/unix-domain-socket-path" yaml:"unixDomainSocket"`
	InternalGRPCPort  int      `arg:"dapr-internal-grpc-port" yaml:"daprInternalGRPCPort" default:"-1"`
}

// SharedRunConfig represents the application configuration parameters, which can be shared across many apps.
type SharedRunConfig struct {
	// Specifically omitted from annotations see https://github.com/dapr/cli/issues/1324
	ConfigFile         string `arg:"config" yaml:"configFilePath"`
	AppProtocol        string `arg:"app-protocol" annotation:"dapr.io/app-protocol" yaml:"appProtocol" default:"http"`
	APIListenAddresses string `arg:"dapr-listen-addresses" annotation:"dapr.io/sidecar-listen-address" yaml:"apiListenAddresses"`
	EnableProfiling    bool   `arg:"enable-profiling" annotation:"dapr.io/enable-profiling" yaml:"enableProfiling"`
	LogLevel           string `arg:"log-level" annotation:"dapr.io.log-level" yaml:"logLevel"`
	MaxConcurrency     int    `arg:"app-max-concurrency" annotation:"dapr.io/app-max-concurrerncy" yaml:"appMaxConcurrency" default:"-1"`
	// Speicifcally omitted from annotations similar to config file path above.
	// Pointer string to distinguish omitted (nil) vs explicitly empty (disable) vs value provided
	PlacementHostAddr *string `arg:"placement-host-address" yaml:"placementHostAddress"`
	// Speicifcally omitted from annotations similar to config file path above.
	ComponentsPath string `arg:"components-path"` // Deprecated in run template file: use ResourcesPaths instead.
	// Speicifcally omitted from annotations similar to config file path above.
	ResourcesPath string `yaml:"resourcesPath"` // Deprecated in run template file: use ResourcesPaths instead.
	// Speicifcally omitted from annotations similar to config file path above.
	ResourcesPaths []string `arg:"resources-path" yaml:"resourcesPaths"`
	// Speicifcally omitted from annotations as appSSL is deprecated.
	AppSSL             bool   `arg:"app-ssl" yaml:"appSSL"`
	MaxRequestBodySize string `arg:"max-body-size" annotation:"dapr.io/max-body-size" yaml:"maxBodySize" default:"4Mi"`
	HTTPReadBufferSize string `arg:"read-buffer-size" annotation:"dapr.io/read-buffer-size" yaml:"readBufferSize" default:"4Ki"`
	EnableAppHealth    bool   `arg:"enable-app-health-check" annotation:"dapr.io/enable-app-health-check" yaml:"enableAppHealthCheck"`
	AppHealthPath      string `arg:"app-health-check-path" annotation:"dapr.io/app-health-check-path" yaml:"appHealthCheckPath"`
	AppHealthInterval  int    `arg:"app-health-probe-interval" annotation:"dapr.io/app-health-probe-interval" ifneq:"0" yaml:"appHealthProbeInterval"`
	AppHealthTimeout   int    `arg:"app-health-probe-timeout" annotation:"dapr.io/app-health-probe-timeout" ifneq:"0" yaml:"appHealthProbeTimeout"`
	AppHealthThreshold int    `arg:"app-health-threshold" annotation:"dapr.io/app-health-threshold" ifneq:"0" yaml:"appHealthThreshold"`
	EnableAPILogging   bool   `arg:"enable-api-logging" annotation:"dapr.io/enable-api-logging" yaml:"enableApiLogging"`
	// Specifically omitted from annotations see https://github.com/dapr/cli/issues/1324 .
	DaprdInstallPath    string            `yaml:"runtimePath"`
	Env                 map[string]string `yaml:"env"`
	DaprdLogDestination LogDestType       `yaml:"daprdLogDestination"`
	AppLogDestination   LogDestType       `yaml:"appLogDestination"`
	// Pointer string to distinguish omitted (nil) vs explicitly empty (disable) vs value provided
	SchedulerHostAddress *string `arg:"scheduler-host-address" yaml:"schedulerHostAddress"`
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
	localLoader := localloader.NewLocalLoader(config.AppID, dirPath)
	err := localLoader.Validate(context.Background())
	if err != nil {
		return fmt.Errorf("error validating components in resources path %q : %w", dirPath, err)
	}
	return nil
}

func (config *RunConfig) validatePlacementHostAddr() error {
	// nil => default localhost:port; empty => disable; non-empty => ensure port
	if config.PlacementHostAddr == nil {
		addr := "localhost"
		if runtime.GOOS == daprWindowsOS {
			addr += ":6050"
		} else {
			addr += ":50005"
		}
		config.PlacementHostAddr = &addr
		return nil
	}
	placementHostAddr := strings.TrimSpace(*config.PlacementHostAddr)
	if len(placementHostAddr) == 0 {
		empty := ""
		config.PlacementHostAddr = &empty
		return nil
	}
	if indx := strings.Index(placementHostAddr, ":"); indx == -1 {
		if runtime.GOOS == daprWindowsOS {
			placementHostAddr += ":6050"
		} else {
			placementHostAddr += ":50005"
		}
	}
	config.PlacementHostAddr = &placementHostAddr
	return nil
}

func (config *RunConfig) validateSchedulerHostAddr() error {
	// nil => leave as-is (set later based on version), empty => disable; non-empty => ensure port
	if config.SchedulerHostAddress == nil {
		return nil
	}
	schedulerHostAddr := strings.TrimSpace(*config.SchedulerHostAddress)
	if len(schedulerHostAddr) == 0 {
		empty := ""
		config.SchedulerHostAddress = &empty
		return nil
	}
	if indx := strings.Index(schedulerHostAddr, ":"); indx == -1 {
		if runtime.GOOS == daprWindowsOS {
			schedulerHostAddr += ":6060"
		} else {
			schedulerHostAddr += ":50006"
		}
	}
	config.SchedulerHostAddress = &schedulerHostAddr
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

	qBody, err := resource.ParseQuantity(config.MaxRequestBodySize)
	if err != nil {
		return fmt.Errorf("invalid max request body size: %w", err)
	}

	if qBody.Value() < 0 {
		config.MaxRequestBodySize = "-1"
	} else {
		config.MaxRequestBodySize = qBody.String()
	}

	qBuffer, err := resource.ParseQuantity(config.HTTPReadBufferSize)
	if err != nil {
		return fmt.Errorf("invalid http read buffer size: %w", err)
	}

	if qBuffer.Value() < 0 {
		config.HTTPReadBufferSize = "-1"
	} else {
		config.HTTPReadBufferSize = qBuffer.String()
	}

	err = config.validatePlacementHostAddr()
	if err != nil {
		return err
	}

	err = config.validateSchedulerHostAddr()
	if err != nil {
		return err
	}
	return nil
}

func (config *RunConfig) ValidateK8s() error {
	meta, err := newDaprMeta()
	if err != nil {
		return err
	}

	if config.AppID == "" {
		config.AppID = meta.newAppID()
	}
	if config.AppPort < 0 {
		config.AppPort = 0
	}
	err = config.validatePort("MetricsPort", &config.MetricsPort, meta)
	if err != nil {
		return err
	}
	if config.MaxConcurrency < 1 {
		config.MaxConcurrency = -1
	}

	qBody, err := resource.ParseQuantity(config.MaxRequestBodySize)
	if err != nil {
		return fmt.Errorf("invalid max request body size: %w", err)
	}

	if qBody.Value() < 0 {
		config.MaxRequestBodySize = "-1"
	} else {
		config.MaxRequestBodySize = qBody.String()
	}

	qBuffer, err := resource.ParseQuantity(config.HTTPReadBufferSize)
	if err != nil {
		return fmt.Errorf("invalid http read buffer size: %w", err)
	}

	if qBuffer.Value() < 0 {
		config.HTTPReadBufferSize = "-1"
	} else {
		config.HTTPReadBufferSize = qBuffer.String()
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
	for i := range schema.NumField() {
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
		case *string:
			if vType != nil {
				val := strings.TrimSpace(*vType)
				if len(val) != 0 && (!hasIfneq || val != ifneq) {
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
	for i := range schema.NumField() {
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

	// Handle values from config that have an "env" tag.
	schema := reflect.ValueOf(*config)
	for i := range schema.NumField() {
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

	// Handle APP_PROTOCOL separately since that requires some additional processing.
	appProtocol := config.getAppProtocol()
	if appProtocol != "" {
		env = append(env, "APP_PROTOCOL="+appProtocol)
	}

	// Add user-defined env vars.
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%v", k, v))
	}

	return env
}

func (config *RunConfig) getAppProtocol() string {
	appProtocol := strings.ToLower(config.AppProtocol)

	switch appProtocol {
	case string("grpcs"), string("https"), string("h2c"):
		return appProtocol
	case string("http"):
		// For backwards compatibility, when protocol is HTTP and --app-ssl is set, use "https".
		if config.AppSSL {
			return "https"
		} else {
			return "http"
		}
	case string("grpc"):
		// For backwards compatibility, when protocol is GRPC and --app-ssl is set, use "grpcs".
		if config.AppSSL {
			return string("grpcs")
		} else {
			return string("grpc")
		}
	case "":
		return string("http")
	default:
		return ""
	}
}

func (config *RunConfig) GetEnv() map[string]string {
	env := map[string]string{}
	schema := reflect.ValueOf(*config)
	for i := range schema.NumField() {
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
		env[key] = value
	}
	for k, v := range config.Env {
		env[k] = v
	}
	return env
}

func (config *RunConfig) GetAnnotations() map[string]string {
	annotations := map[string]string{}
	schema := reflect.ValueOf(*config)
	for i := range schema.NumField() {
		valueField := schema.Field(i).Interface()
		typeField := schema.Type().Field(i)
		key := typeField.Tag.Get("annotation")
		if len(key) == 0 {
			continue
		}
		if value, ok := valueField.(int); ok && value <= 0 {
			// ignore unset numeric variables.
			continue
		}
		value := fmt.Sprintf("%v", reflect.ValueOf(valueField))
		annotations[key] = value
	}
	return annotations
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

	var cmd *exec.Cmd
	if runtime.GOOS == daprWindowsOS {
		// On Windows, run the executable directly (no shell).
		// TODO: In future this will likely need updates if Windows faces the same Python threading issues.
		cmd = exec.Command(command, args...)
	} else {
		// Use shell exec to avoid forking, which breaks Python threading on Unix
		shArgs := []string{"-c", "exec \"$@\"", "sh", command}
		shArgs = append(shArgs, args...)
		cmd = exec.Command("sh", shArgs...)
	}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, config.getEnv()...)
	setProcessGroup(cmd)

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
