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
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func setupRun(t *testing.T) {
	componentsDir := DefaultComponentsDirPath()
	configFile := DefaultConfigFilePath()
	err := os.MkdirAll(componentsDir, 0o700)
	assert.Equal(t, nil, err, "Unable to setup components dir before running test")
	file, err := os.Create(configFile)
	file.Close()
	assert.Equal(t, nil, err, "Unable to create config file before running test")
}

func tearDownRun(t *testing.T) {
	err := os.RemoveAll(DefaultComponentsDirPath())
	assert.Equal(t, nil, err, "Unable to delete default components dir after running test")
	err = os.Remove(DefaultConfigFilePath())
	assert.Equal(t, nil, err, "Unable to delete default config file after running test")
}

func assertCommonArgs(t *testing.T, basicConfig *RunConfig, output *RunOutput) {
	assert.NotNil(t, output)

	assert.Equal(t, "MyID", output.AppID)
	assert.Equal(t, 8000, output.DaprHTTPPort)
	assert.Equal(t, 50001, output.DaprGRPCPort)

	assert.Contains(t, output.DaprCMD.Args[0], "daprd")
	assertArgumentEqual(t, "app-id", "MyID", output.DaprCMD.Args)
	assertArgumentEqual(t, "dapr-http-port", "8000", output.DaprCMD.Args)
	assertArgumentEqual(t, "dapr-grpc-port", "50001", output.DaprCMD.Args)
	assertArgumentEqual(t, "log-level", basicConfig.LogLevel, output.DaprCMD.Args)
	assertArgumentEqual(t, "app-max-concurrency", "-1", output.DaprCMD.Args)
	assertArgumentEqual(t, "app-protocol", "http", output.DaprCMD.Args)
	assertArgumentEqual(t, "app-port", "3000", output.DaprCMD.Args)
	assertArgumentEqual(t, "components-path", DefaultComponentsDirPath(), output.DaprCMD.Args)
	assertArgumentEqual(t, "app-ssl", "", output.DaprCMD.Args)
	assertArgumentEqual(t, "metrics-port", "9001", output.DaprCMD.Args)
	assertArgumentEqual(t, "dapr-http-max-request-size", "-1", output.DaprCMD.Args)
	assertArgumentEqual(t, "dapr-http-read-buffer-size", "-1", output.DaprCMD.Args)
}

func assertAppEnv(t *testing.T, config *RunConfig, output *RunOutput) {
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

func getEnvSet(config *RunConfig) []string {
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

	basicConfig := &RunConfig{
		AppID:              "MyID",
		AppPort:            3000,
		HTTPPort:           8000,
		GRPCPort:           50001,
		LogLevel:           "WARN",
		Arguments:          []string{"MyCommand", "--my-arg"},
		EnableProfiling:    false,
		ProfilePort:        9090,
		Protocol:           "http",
		ComponentsPath:     DefaultComponentsDirPath(),
		AppSSL:             true,
		MetricsPort:        9001,
		MaxRequestBodySize: -1,
		HTTPReadBufferSize: -1,
		EnableAPILogging:   true,
	}

	t.Run("run happy http", func(t *testing.T) {
		output, err := Run(basicConfig)
		assert.Nil(t, err)

		assertCommonArgs(t, basicConfig, output)
		assert.Equal(t, "MyCommand", output.AppCMD.Args[0])
		assert.Equal(t, "--my-arg", output.AppCMD.Args[1])
		assertAppEnv(t, basicConfig, output)
	})

	t.Run("run without app command", func(t *testing.T) {
		basicConfig.Arguments = nil
		basicConfig.LogLevel = "INFO"
		basicConfig.EnableAPILogging = true
		basicConfig.ConfigFile = DefaultConfigFilePath()
		output, err := Run(basicConfig)
		assert.Nil(t, err)

		assertCommonArgs(t, basicConfig, output)
		assertArgumentEqual(t, "config", DefaultConfigFilePath(), output.DaprCMD.Args)
		assert.Nil(t, output.AppCMD)
	})

	t.Run("run without port", func(t *testing.T) {
		basicConfig.HTTPPort = -1
		basicConfig.GRPCPort = -1
		basicConfig.MetricsPort = -1
		output, err := Run(basicConfig)

		assert.Nil(t, err)
		assert.NotNil(t, output)

		assertArgumentNotEqual(t, "http-port", "-1", output.DaprCMD.Args)
		assertArgumentNotEqual(t, "grpc-port", "-1", output.DaprCMD.Args)
		assertArgumentNotEqual(t, "metrics-port", "-1", output.DaprCMD.Args)
	})
}
