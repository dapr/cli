/*
Copyright 2023 The Dapr Authors
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

package runexec

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/dapr/cli/pkg/standalone"
)

const windowsOsType = "windows"

func assertArgumentEqual(t *testing.T, key string, expectedValue string, args []string) {
	var value string
	for index, arg := range args {
		if arg == "--"+key {
			nextIndex := index + 1
			if nextIndex < len(args) {
				if !strings.HasPrefix(args[nextIndex], "--") {
					value = args[nextIndex]
				}
			}
		}
	}

	assert.Equal(t, expectedValue, value)
}

func assertArgumentNotEqual(t *testing.T, key string, expectedValue string, args []string) {
	var value string
	for index, arg := range args {
		if arg == "--"+key {
			nextIndex := index + 1
			if nextIndex < len(args) {
				if !strings.HasPrefix(args[nextIndex], "--") {
					value = args[nextIndex]
				}
			}
		}
	}

	assert.NotEqual(t, expectedValue, value)
}

func assertArgumentContains(t *testing.T, key string, expectedValue string, args []string) {
	var value string
	for index, arg := range args {
		if arg == "--"+key {
			nextIndex := index + 1
			if nextIndex < len(args) {
				if !strings.HasPrefix(args[nextIndex], "--") {
					value = args[nextIndex]
				}
			}
		}
	}

	assert.Contains(t, value, expectedValue)
}

func setupRun(t *testing.T) {
	myDaprPath, err := standalone.GetDaprRuntimePath("")
	assert.NoError(t, err)

	componentsDir := standalone.GetDaprComponentsPath(myDaprPath)
	configFile := standalone.GetDaprConfigPath(myDaprPath)
	err = os.MkdirAll(componentsDir, 0o700)
	assert.NoError(t, err, "Unable to setup components dir before running test")
	file, err := os.Create(configFile)
	file.Close()
	assert.NoError(t, err, "Unable to create config file before running test")
}

func tearDownRun(t *testing.T) {
	myDaprPath, err := standalone.GetDaprRuntimePath("")
	assert.NoError(t, err)

	componentsDir := standalone.GetDaprComponentsPath(myDaprPath)
	configFile := standalone.GetDaprConfigPath(myDaprPath)

	err = os.RemoveAll(componentsDir)
	assert.NoError(t, err, "Unable to delete default components dir after running test")
	err = os.Remove(configFile)
	assert.NoError(t, err, "Unable to delete default config file after running test")
}

func assertCommonArgs(t *testing.T, basicConfig *standalone.RunConfig, output *RunOutput) {
	assert.NotNil(t, output)

	assert.Equal(t, "MyID", output.AppID)
	assert.Equal(t, 8000, output.DaprHTTPPort)
	assert.Equal(t, 50001, output.DaprGRPCPort)

	daprPath, err := standalone.GetDaprRuntimePath("")
	assert.NoError(t, err)

	assert.Contains(t, output.DaprCMD.Args[0], "daprd")
	assertArgumentEqual(t, "app-id", "MyID", output.DaprCMD.Args)
	assertArgumentEqual(t, "dapr-http-port", "8000", output.DaprCMD.Args)
	assertArgumentEqual(t, "dapr-grpc-port", "50001", output.DaprCMD.Args)
	assertArgumentEqual(t, "log-level", basicConfig.LogLevel, output.DaprCMD.Args)
	assertArgumentEqual(t, "app-max-concurrency", "-1", output.DaprCMD.Args)
	assertArgumentEqual(t, "app-protocol", "http", output.DaprCMD.Args)
	assertArgumentEqual(t, "app-port", "3000", output.DaprCMD.Args)
	assertArgumentEqual(t, "components-path", standalone.GetDaprComponentsPath(daprPath), output.DaprCMD.Args)
	assertArgumentEqual(t, "app-ssl", "", output.DaprCMD.Args)
	assertArgumentEqual(t, "metrics-port", "9001", output.DaprCMD.Args)
	assertArgumentEqual(t, "max-body-size", "-1", output.DaprCMD.Args)
	assertArgumentEqual(t, "dapr-internal-grpc-port", "5050", output.DaprCMD.Args)
	assertArgumentEqual(t, "read-buffer-size", "-1", output.DaprCMD.Args)
	assertArgumentEqual(t, "dapr-listen-addresses", "127.0.0.1", output.DaprCMD.Args)
}

func assertAppEnv(t *testing.T, config *standalone.RunConfig, output *RunOutput) {
	envSet := make(map[string]bool)
	for _, env := range output.AppCMD.Env {
		envSet[env] = true
	}

	expectedEnvSet := getEnvSet(config)
	for _, env := range expectedEnvSet {
		_, found := envSet[env]
		if !found {
			assert.Fail(t, "Missing environment variable. Expected to have "+env)
		}
	}
}

func getEnvSet(config *standalone.RunConfig) []string {
	set := []string{
		getEnv("DAPR_GRPC_PORT", config.GRPCPort),
		getEnv("DAPR_HTTP_PORT", config.HTTPPort),
		getEnv("DAPR_METRICS_PORT", config.MetricsPort),
		getEnv("APP_ID", config.AppID),
	}
	if config.AppPort > 0 {
		set = append(set, getEnv("APP_PORT", config.AppPort))
	}
	if config.EnableProfiling {
		set = append(set, getEnv("DAPR_PROFILE_PORT", config.ProfilePort))
	}
	return set
}

func getEnv(key string, value interface{}) string {
	return fmt.Sprintf("%s=%v", key, value)
}

func TestRun(t *testing.T) {
	// Setup the components directory which is done at init time.
	setupRun(t)

	// Setup the tearDown routine to run in the end.
	defer tearDownRun(t)

	myDaprPath, err := standalone.GetDaprRuntimePath("")
	assert.NoError(t, err)

	componentsDir := standalone.GetDaprComponentsPath(myDaprPath)
	configFile := standalone.GetDaprConfigPath(myDaprPath)

	sharedRunConfig := &standalone.SharedRunConfig{
		LogLevel:           "WARN",
		EnableProfiling:    false,
		AppProtocol:        "http",
		ComponentsPath:     componentsDir,
		AppSSL:             true,
		MaxRequestBodySize: "-1",
		HTTPReadBufferSize: "-1",
		EnableAPILogging:   true,
		APIListenAddresses: "127.0.0.1",
	}
	basicConfig := &standalone.RunConfig{
		AppID:             "MyID",
		AppPort:           3000,
		HTTPPort:          8000,
		GRPCPort:          50001,
		Command:           []string{"MyCommand", "--my-arg"},
		ProfilePort:       9090,
		MetricsPort:       9001,
		InternalGRPCPort:  5050,
		AppChannelAddress: "localhost",
		SharedRunConfig:   *sharedRunConfig,
	}

	t.Run("run happy http", func(t *testing.T) {
		output, err := NewOutput(basicConfig)
		assert.NoError(t, err)

		assertCommonArgs(t, basicConfig, output)
		require.NotNil(t, output.AppCMD)
		if runtime.GOOS == windowsOsType {
			// On Windows the app is run directly (no shell).
			require.GreaterOrEqual(t, len(output.AppCMD.Args), 2)
			assert.Equal(t, "MyCommand", output.AppCMD.Args[0])
			assert.Equal(t, "--my-arg", output.AppCMD.Args[1])
		} else {
			// On Unix the app command is executed via a shell wrapper
			require.GreaterOrEqual(t, len(output.AppCMD.Args), 5)
			assert.Equal(t, "sh", output.AppCMD.Args[0])
			assert.Equal(t, "-c", output.AppCMD.Args[1])
			assert.Equal(t, "exec \"$@\"", output.AppCMD.Args[2])
			assert.Equal(t, "sh", output.AppCMD.Args[3])
			assert.Equal(t, "MyCommand", output.AppCMD.Args[4])
			assert.Equal(t, "--my-arg", output.AppCMD.Args[5])
		}

		assertArgumentEqual(t, "app-channel-address", "localhost", output.DaprCMD.Args)
		assertAppEnv(t, basicConfig, output)
	})

	t.Run("run without app command", func(t *testing.T) {
		basicConfig.Command = nil
		basicConfig.LogLevel = "INFO"
		basicConfig.EnableAPILogging = true
		basicConfig.ConfigFile = configFile
		output, err := NewOutput(basicConfig)
		assert.NoError(t, err)

		assertCommonArgs(t, basicConfig, output)
		assertArgumentContains(t, "config", standalone.DefaultConfigFileName, output.DaprCMD.Args)
		assert.Nil(t, output.AppCMD)
	})

	t.Run("run without port", func(t *testing.T) {
		basicConfig.HTTPPort = -1
		basicConfig.GRPCPort = -1
		basicConfig.MetricsPort = -1
		output, err := NewOutput(basicConfig)

		assert.NoError(t, err)
		assert.NotNil(t, output)

		assertArgumentNotEqual(t, "http-port", "-1", output.DaprCMD.Args)
		assertArgumentNotEqual(t, "grpc-port", "-1", output.DaprCMD.Args)
		assertArgumentNotEqual(t, "metrics-port", "-1", output.DaprCMD.Args)
	})

	t.Run("app health check flags missing if not set", func(t *testing.T) {
		output, err := NewOutput(basicConfig)

		assert.NoError(t, err)
		assert.NotNil(t, output)

		argsFlattened := strings.Join(output.DaprCMD.Args, " ")
		assert.NotRegexp(t, regexp.MustCompile(`( |^)--enable-app-health-check( |$)`), argsFlattened)
		assert.NotRegexp(t, regexp.MustCompile(`( |^)--app-health-check-path( |=)`), argsFlattened)
		assert.NotRegexp(t, regexp.MustCompile(`( |^)--app-health-probe-interval( |=)`), argsFlattened)
		assert.NotRegexp(t, regexp.MustCompile(`( |^)--app-health-probe-timeout( |=)`), argsFlattened)
		assert.NotRegexp(t, regexp.MustCompile(`( |^)--app-health-threshold( |=)`), argsFlattened)
	})

	t.Run("enable app health checks with default flags", func(t *testing.T) {
		basicConfig.EnableAppHealth = true
		output, err := NewOutput(basicConfig)

		assert.NoError(t, err)
		assert.NotNil(t, output)

		argsFlattened := strings.Join(output.DaprCMD.Args, " ")
		assert.Regexp(t, regexp.MustCompile(`( |^)--enable-app-health-check( |$)`), argsFlattened)

		// Other flags are not included so daprd can use the default value.
		assert.NotRegexp(t, regexp.MustCompile(`( |^)--app-health-check-path( |=)`), argsFlattened)
		assert.NotRegexp(t, regexp.MustCompile(`( |^)--app-health-probe-interval( |=)`), argsFlattened)
		assert.NotRegexp(t, regexp.MustCompile(`( |^)--app-health-probe-timeout( |=)`), argsFlattened)
		assert.NotRegexp(t, regexp.MustCompile(`( |^)--app-health-threshold( |=)`), argsFlattened)
	})

	t.Run("enable app health checks with all flags set", func(t *testing.T) {
		basicConfig.EnableAppHealth = true
		basicConfig.AppHealthInterval = 2
		basicConfig.AppHealthTimeout = 200
		basicConfig.AppHealthThreshold = 1
		basicConfig.AppHealthPath = "/foo"
		output, err := NewOutput(basicConfig)

		assert.NoError(t, err)
		assert.NotNil(t, output)

		argsFlattened := strings.Join(output.DaprCMD.Args, " ")
		assert.Regexp(t, regexp.MustCompile(`( |^)--enable-app-health-check( |$)`), argsFlattened)
		assert.Regexp(t, regexp.MustCompile(`( |^)--app-health-check-path( |=)/foo`), argsFlattened)
		assert.Regexp(t, regexp.MustCompile(`( |^)--app-health-probe-interval( |=)2`), argsFlattened)
		assert.Regexp(t, regexp.MustCompile(`( |^)--app-health-probe-timeout( |=)200`), argsFlattened)
		assert.Regexp(t, regexp.MustCompile(`( |^)--app-health-threshold( |=)1`), argsFlattened)
	})

	t.Run("test setting defaults from struct tag", func(t *testing.T) {
		basicConfig.AppPort = 0
		basicConfig.HTTPPort = 0
		basicConfig.GRPCPort = 0
		basicConfig.MetricsPort = 0
		basicConfig.ProfilePort = 0
		basicConfig.EnableProfiling = true
		basicConfig.MaxConcurrency = 0
		basicConfig.MaxRequestBodySize = ""
		basicConfig.HTTPReadBufferSize = ""
		basicConfig.AppProtocol = ""

		basicConfig.SetDefaultFromSchema()

		assert.Equal(t, -1, basicConfig.AppPort)
		assert.Equal(t, -1, basicConfig.HTTPPort)
		assert.Equal(t, -1, basicConfig.GRPCPort)
		assert.Equal(t, -1, basicConfig.MetricsPort)
		assert.Equal(t, -1, basicConfig.ProfilePort)
		assert.True(t, basicConfig.EnableProfiling)
		assert.Equal(t, -1, basicConfig.MaxConcurrency)
		assert.Equal(t, "4Mi", basicConfig.MaxRequestBodySize)
		assert.Equal(t, "4Ki", basicConfig.HTTPReadBufferSize)
		assert.Equal(t, "http", basicConfig.AppProtocol)

		// Test after Validate gets called.
		err := basicConfig.Validate()
		assert.NoError(t, err)

		assert.Equal(t, 0, basicConfig.AppPort)
		assert.Positive(t, basicConfig.HTTPPort)
		assert.Positive(t, basicConfig.GRPCPort)
		assert.Positive(t, basicConfig.MetricsPort)
		assert.Positive(t, basicConfig.ProfilePort)
		assert.True(t, basicConfig.EnableProfiling)
		assert.Equal(t, -1, basicConfig.MaxConcurrency)
		assert.Equal(t, "4Mi", basicConfig.MaxRequestBodySize)
		assert.Equal(t, "4Ki", basicConfig.HTTPReadBufferSize)
		assert.Equal(t, "http", basicConfig.AppProtocol)
	})

	t.Run("run with max body size without units", func(t *testing.T) {
		basicConfig.MaxRequestBodySize = "4000000"

		output, err := NewOutput(basicConfig)
		require.NoError(t, err)
		assertArgumentEqual(t, "max-body-size", "4M", output.DaprCMD.Args)
	})

	t.Run("run with max body size with units", func(t *testing.T) {
		basicConfig.MaxRequestBodySize = "4Mi"

		output, err := NewOutput(basicConfig)
		require.NoError(t, err)
		assertArgumentEqual(t, "max-body-size", "4Mi", output.DaprCMD.Args)

		basicConfig.MaxRequestBodySize = "5M"

		output, err = NewOutput(basicConfig)
		require.NoError(t, err)
		assertArgumentEqual(t, "max-body-size", "5M", output.DaprCMD.Args)
	})

	t.Run("run with read buffer size set without units", func(t *testing.T) {
		basicConfig.HTTPReadBufferSize = "16001"

		output, err := NewOutput(basicConfig)
		require.NoError(t, err)
		assertArgumentEqual(t, "read-buffer-size", "16001", output.DaprCMD.Args)
	})

	t.Run("run with read buffer size set with units", func(t *testing.T) {
		basicConfig.HTTPReadBufferSize = "4Ki"

		output, err := NewOutput(basicConfig)
		require.NoError(t, err)
		assertArgumentEqual(t, "read-buffer-size", "4Ki", output.DaprCMD.Args)
	})
}
