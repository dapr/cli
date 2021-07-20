// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	daprEnv "github.com/dapr/dapr/pkg/config/env"
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
	err := os.MkdirAll(componentsDir, 0700)
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

	placementHostAddr := "localhost:50005"
	if runtime.GOOS == "windows" {
		placementHostAddr = "localhost:6050"
	}
	assertArgumentEqual(t, "placement-host-address", placementHostAddr, output.DaprCMD.Args)
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
		getEnv(daprEnv.DaprGRPCPort, config.GRPCPort),
		getEnv(daprEnv.DaprHTTPPort, config.HTTPPort),
		getEnv(daprEnv.DaprMetricsPort, config.MetricsPort),
		getEnv(daprEnv.AppID, config.AppID),
		getEnv(daprEnv.AppPort, config.AppPort),
	}
	if config.EnableProfiling {
		set = append(set, getEnv(daprEnv.DaprProfilePort, config.ProfilePort))
	}
	return set
}

func getEnv(key string, value interface{}) string {
	return fmt.Sprintf("%s=%v", key, value)
}

func TestRun(t *testing.T) {
	// Setup the components directory which is done at init time
	setupRun(t)

	// Setup the tearDown routine to run in the end
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
		PlacementHostAddr:  "localhost",
		ComponentsPath:     DefaultComponentsDirPath(),
		AppSSL:             true,
		MetricsPort:        9001,
		MaxRequestBodySize: -1,
	}

	t.Run("run happy http", func(t *testing.T) {
		output, err := Run(basicConfig)
		assert.Nil(t, err)
		assert.NotNil(t, output)

		assert.Equal(t, "MyID", output.AppID)
		assert.Equal(t, 8000, output.DaprHTTPPort)
		assert.Equal(t, 50001, output.DaprGRPCPort)

		assert.Contains(t, output.DaprCMD.Args[0], "daprd")
		assertArgumentEqual(t, "app-id", "MyID", output.DaprCMD.Args)
		assertArgumentEqual(t, "dapr-http-port", "8000", output.DaprCMD.Args)
		assertArgumentEqual(t, "dapr-grpc-port", "50001", output.DaprCMD.Args)
		assertArgumentEqual(t, "log-level", "WARN", output.DaprCMD.Args)
		assertArgumentEqual(t, "app-max-concurrency", "-1", output.DaprCMD.Args)
		assertArgumentEqual(t, "app-protocol", "http", output.DaprCMD.Args)
		assertArgumentEqual(t, "app-port", "3000", output.DaprCMD.Args)
		assertArgumentEqual(t, "components-path", DefaultComponentsDirPath(), output.DaprCMD.Args)
		assertArgumentEqual(t, "app-ssl", "", output.DaprCMD.Args)
		assertArgumentEqual(t, "metrics-port", "9001", output.DaprCMD.Args)
		if runtime.GOOS == "windows" {
			assertArgumentEqual(t, "placement-host-address", "localhost:6050", output.DaprCMD.Args)
		} else {
			assertArgumentEqual(t, "placement-host-address", "localhost:50005", output.DaprCMD.Args)
		}
		assertArgumentEqual(t, "dapr-http-max-request-size", "-1", output.DaprCMD.Args)

		assertCommonArgs(t, basicConfig, output)
		assert.Equal(t, "MyCommand", output.AppCMD.Args[0])
		assert.Equal(t, "--my-arg", output.AppCMD.Args[1])
		assertAppEnv(t, basicConfig, output)
	})

	t.Run("run without app command", func(t *testing.T) {
		basicConfig.Arguments = nil
		basicConfig.LogLevel = "INFO"
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

	t.Run("run with specified placement-host port", func(t *testing.T) {
		basicConfig.PlacementHostAddr = "localhost:12345"
		output, err := Run(basicConfig)

		assert.Nil(t, err)
		assert.NotNil(t, output)

		assertArgumentEqual(t, "placement-host-address", "localhost:12345", output.DaprCMD.Args)
	})
}
