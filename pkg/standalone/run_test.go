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
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	config := &RunConfig{
		SharedRunConfig:   SharedRunConfig{},
		AppID:             "testapp",
		AppChannelAddress: "localhost",
		AppPort:           1234,
		HTTPPort:          2345,
		GRPCPort:          3456,
		ProfilePort:       4567, // This is not included in env.
		MetricsPort:       5678,
	}

	t.Run("no explicit app-protocol", func(t *testing.T) {
		expect := []string{
			"APP_ID=testapp",
			"APP_CHANNEL_ADDRESS=localhost",
			"APP_PORT=1234",
			"APP_PROTOCOL=http",
			"DAPR_HTTP_PORT=2345",
			"DAPR_GRPC_PORT=3456",
			"DAPR_METRICS_PORT=5678",
		}

		got := config.getEnv()

		sort.Strings(expect)
		sort.Strings(got)

		assert.Equal(t, expect, got)
	})

	t.Run("app-protocol grpcs", func(t *testing.T) {
		config.AppProtocol = "grpcs"
		config.AppSSL = false

		expect := []string{
			"APP_ID=testapp",
			"APP_CHANNEL_ADDRESS=localhost",
			"APP_PORT=1234",
			"APP_PROTOCOL=grpcs",
			"DAPR_HTTP_PORT=2345",
			"DAPR_GRPC_PORT=3456",
			"DAPR_METRICS_PORT=5678",
		}

		got := config.getEnv()

		sort.Strings(expect)
		sort.Strings(got)

		assert.Equal(t, expect, got)
	})

	t.Run("app-protocol http", func(t *testing.T) {
		config.AppProtocol = "http"
		config.AppSSL = false

		expect := []string{
			"APP_ID=testapp",
			"APP_CHANNEL_ADDRESS=localhost",
			"APP_PORT=1234",
			"APP_PROTOCOL=http",
			"DAPR_HTTP_PORT=2345",
			"DAPR_GRPC_PORT=3456",
			"DAPR_METRICS_PORT=5678",
		}

		got := config.getEnv()

		sort.Strings(expect)
		sort.Strings(got)

		assert.Equal(t, expect, got)
	})

	t.Run("app-protocol http with app-ssl", func(t *testing.T) {
		config.AppProtocol = "http"
		config.AppSSL = true

		expect := []string{
			"APP_ID=testapp",
			"APP_CHANNEL_ADDRESS=localhost",
			"APP_PORT=1234",
			"APP_PROTOCOL=https",
			"DAPR_HTTP_PORT=2345",
			"DAPR_GRPC_PORT=3456",
			"DAPR_METRICS_PORT=5678",
		}

		got := config.getEnv()

		sort.Strings(expect)
		sort.Strings(got)

		assert.Equal(t, expect, got)
	})

	t.Run("app-protocol grpc with app-ssl", func(t *testing.T) {
		config.AppProtocol = "grpc"
		config.AppSSL = true

		expect := []string{
			"APP_ID=testapp",
			"APP_CHANNEL_ADDRESS=localhost",
			"APP_PORT=1234",
			"APP_PROTOCOL=grpcs",
			"DAPR_HTTP_PORT=2345",
			"DAPR_GRPC_PORT=3456",
			"DAPR_METRICS_PORT=5678",
		}

		got := config.getEnv()

		sort.Strings(expect)
		sort.Strings(got)

		assert.Equal(t, expect, got)
	})
}

func TestValidatePlacementHostAddr(t *testing.T) {
	t.Run("empty disables placement", func(t *testing.T) {
		cfg := &RunConfig{SharedRunConfig: SharedRunConfig{PlacementHostAddr: ""}}
		err := cfg.validatePlacementHostAddr()
		assert.NoError(t, err)
		assert.Equal(t, "", cfg.PlacementHostAddr)
	})

	t.Run("default port appended when hostname provided without port", func(t *testing.T) {
		cfg := &RunConfig{SharedRunConfig: SharedRunConfig{PlacementHostAddr: "localhost"}}
		err := cfg.validatePlacementHostAddr()
		assert.NoError(t, err)
		if runtime.GOOS == daprWindowsOS {
			assert.True(t, strings.HasSuffix(cfg.PlacementHostAddr, ":6050"))
		} else {
			assert.True(t, strings.HasSuffix(cfg.PlacementHostAddr, ":50005"))
		}
	})

	t.Run("custom port preserved when provided", func(t *testing.T) {
		cfg := &RunConfig{SharedRunConfig: SharedRunConfig{PlacementHostAddr: "1.2.3.4:12345"}}
		err := cfg.validatePlacementHostAddr()
		assert.NoError(t, err)
		assert.Equal(t, "1.2.3.4:12345", cfg.PlacementHostAddr)
	})
}
