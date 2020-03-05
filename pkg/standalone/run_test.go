// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertArgument(t *testing.T, key string, expectedValue string, args []string) {
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

func TestRun(t *testing.T) {
	t.Run("run happy http", func(t *testing.T) {
		output, err := Run(&RunConfig{
			AppID:           "MyID",
			AppPort:         3000,
			HTTPPort:        8000,
			GRPCPort:        50001,
			LogLevel:        "WARN",
			Arguments:       []string{"MyCommand", "--my-arg"},
			EnableProfiling: false,
			ProfilePort:     9090,
			Protocol:        "http",
			RedisHost:       "localhost",
			PlacementHost:   "localhost",
		})

		assert.Nil(t, err)
		assert.NotNil(t, output)

		assert.Equal(t, "MyID", output.AppID)
		assert.Equal(t, 8000, output.DaprHTTPPort)
		assert.Equal(t, 50001, output.DaprGRPCPort)

		assert.Contains(t, output.DaprCMD.Args[0], "daprd")
		assertArgument(t, "app-id", "MyID", output.DaprCMD.Args)
		assertArgument(t, "dapr-http-port", "8000", output.DaprCMD.Args)
		assertArgument(t, "dapr-grpc-port", "50001", output.DaprCMD.Args)
		assertArgument(t, "log-level", "WARN", output.DaprCMD.Args)
		assertArgument(t, "max-concurrency", "-1", output.DaprCMD.Args)
		assertArgument(t, "protocol", "http", output.DaprCMD.Args)
		assertArgument(t, "app-port", "3000", output.DaprCMD.Args)
		if runtime.GOOS == "windows" {
			assertArgument(t, "placement-address", "localhost:6050", output.DaprCMD.Args)
		} else {
			assertArgument(t, "placement-address", "localhost:50005", output.DaprCMD.Args)
		}

		assert.Equal(t, "MyCommand", output.AppCMD.Args[0])
		assert.Equal(t, "--my-arg", output.AppCMD.Args[1])
	})

	t.Run("run without app command", func(t *testing.T) {
		output, err := Run(&RunConfig{
			AppID:           "MyID",
			AppPort:         3000,
			HTTPPort:        8000,
			GRPCPort:        50001,
			LogLevel:        "INFO",
			EnableProfiling: false,
			ProfilePort:     9090,
			Protocol:        "http",
			RedisHost:       "localhost",
			PlacementHost:   "localhost",
		})

		assert.Nil(t, err)
		assert.NotNil(t, output)

		assert.Equal(t, "MyID", output.AppID)
		assert.Equal(t, 8000, output.DaprHTTPPort)
		assert.Equal(t, 50001, output.DaprGRPCPort)

		assert.Contains(t, output.DaprCMD.Args[0], "daprd")
		assertArgument(t, "app-id", "MyID", output.DaprCMD.Args)
		assertArgument(t, "dapr-http-port", "8000", output.DaprCMD.Args)
		assertArgument(t, "dapr-grpc-port", "50001", output.DaprCMD.Args)
		assertArgument(t, "log-level", "INFO", output.DaprCMD.Args)
		assertArgument(t, "max-concurrency", "-1", output.DaprCMD.Args)
		assertArgument(t, "protocol", "http", output.DaprCMD.Args)
		assertArgument(t, "app-port", "3000", output.DaprCMD.Args)
		if runtime.GOOS == "windows" {
			assertArgument(t, "placement-address", "localhost:6050", output.DaprCMD.Args)
		} else {
			assertArgument(t, "placement-address", "localhost:50005", output.DaprCMD.Args)
		}

		assert.Nil(t, output.AppCMD)
	})
}
