
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
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testMTLSEnabledConfig = `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprsystem
spec:
  mtls:
    enabled: true
`

func strPtr(s string) *string { return &s }

// setupMTLSTest creates an isolated Dapr installation directory.
func setupMTLSTest(t *testing.T) string {
	t.Helper()

	runtimePath := t.TempDir()

	daprDir, err := GetDaprRuntimePath(runtimePath)
	assert.NoError(t, err)

	return daprDir
}

// writeMTLSConfig creates a Dapr configuration file in the installation directory.
func writeMTLSConfig(t *testing.T, daprDir, content string) string {
	t.Helper()

	assert.NoError(t, os.MkdirAll(daprDir, 0o755))

	configFile := GetDaprConfigPath(daprDir)
	assert.NoError(t, os.WriteFile(configFile, []byte(content), 0o600))

	return configFile
}

// writeMTLSCerts creates certificates in the same directory structure
// used by the Sentry installation.
func writeMTLSCerts(t *testing.T, daprDir string) {
	t.Helper()

	certsDir := GetDaprSentryCertsPath(daprDir)
	assert.NoError(t, os.MkdirAll(certsDir, 0o755))

	for _, cert := range []struct {
		name    string
		content string
	}{
		{
			name:    "ca.crt",
			content: "content of ca.crt",
		},
		{
			name:    "issuer.crt",
			content: "content of issuer.crt",
		},
		{
			name:    "issuer.key",
			content: "content of issuer.key",
		},
	} {
		assert.NoError(t, os.WriteFile(
			filepath.Join(certsDir, cert.name),
			[]byte(cert.content),
			0o600,
		))
	}
}

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

func TestGetDaprCommandEnv(t *testing.T) {
	t.Run("no user-defined env vars", func(t *testing.T) {
		config := &RunConfig{AppID: "testapp"}

		cmd, err := GetDaprCommand(config)

		assert.NoError(t, err)
		assert.Nil(t, cmd.Env)
	})

	t.Run("user-defined env vars are passed to daprd", func(t *testing.T) {
		config := &RunConfig{
			AppID: "testapp",
			SharedRunConfig: SharedRunConfig{
				Env: map[string]string{
					"MYAPP_API_PORT": "3000",
				},
			},
		}

		cmd, err := GetDaprCommand(config)

		assert.NoError(t, err)
		assert.Contains(t, cmd.Env, "MYAPP_API_PORT=3000")
	})

	t.Run("mTLS enabled adds daprd arguments", func(t *testing.T) {
		daprDir := setupMTLSTest(t)
		configFile := writeMTLSConfig(t, daprDir, testMTLSEnabledConfig)

		config := &RunConfig{
			AppID: "testapp",
			SharedRunConfig: SharedRunConfig{
				ConfigFile: configFile,
			},
		}

		cmd, err := GetDaprCommand(config)

		assert.NoError(t, err)

		assert.Contains(t, cmd.Args, "--enable-mtls")
		assert.Contains(t, cmd.Args, "--sentry-address")
		assert.Contains(t, cmd.Args, sentryDefaultAddress)
	})

	t.Run("mTLS disabled adds no mTLS arguments", func(t *testing.T) {
		daprDir := setupMTLSTest(t)
		configFile := writeMTLSConfig(t, daprDir, "spec: {}\n")

		config := &RunConfig{
			AppID: "testapp",
			SharedRunConfig: SharedRunConfig{
				ConfigFile: configFile,
			},
		}

		cmd, err := GetDaprCommand(config)

		assert.NoError(t, err)

		assert.NotContains(t, cmd.Args, "--enable-mtls")
		assert.NotContains(t, cmd.Args, "--sentry-address")
	})
}

func TestGetAppCommandMTLSEnv(t *testing.T) {
	daprDir := setupMTLSTest(t)
	writeMTLSCerts(t, daprDir)

	// MTLSEnv accepts the Dapr installation directory, not the certs directory.
	mtlsEnv, err := MTLSEnv(daprDir)
	assert.NoError(t, err)

	assert.Equal(t, map[string]string{
		EnvDaprTrustAnchors: "content of ca.crt",
		EnvDaprCertChain:    "content of issuer.crt",
		EnvDaprCertKey:      "content of issuer.key",
	}, mtlsEnv)

	config := &RunConfig{
		AppID:   "testapp",
		Command: []string{"echo", "hello"},
		SharedRunConfig: SharedRunConfig{
			Env: mtlsEnv,
		},
	}

	cmd := GetAppCommand(config)

	assert.NotNil(t, cmd)

	assert.Contains(t, cmd.Env, EnvDaprTrustAnchors+"=content of ca.crt")
	assert.Contains(t, cmd.Env, EnvDaprCertChain+"=content of issuer.crt")
	assert.Contains(t, cmd.Env, EnvDaprCertKey+"=content of issuer.key")
}

func TestValidatePlacementHostAddr(t *testing.T) {
	t.Run("empty disables placement", func(t *testing.T) {
		cfg := &RunConfig{SharedRunConfig: SharedRunConfig{PlacementHostAddr: strPtr("")}}
		err := cfg.validatePlacementHostAddr()
		assert.NoError(t, err)
		assert.NotNil(t, cfg.PlacementHostAddr)
		assert.Equal(t, "", *cfg.PlacementHostAddr)
	})

	t.Run("whitespace disables placement", func(t *testing.T) {
		cfg := &RunConfig{SharedRunConfig: SharedRunConfig{PlacementHostAddr: strPtr("  ")}}
		err := cfg.validatePlacementHostAddr()
		assert.NoError(t, err)
		assert.NotNil(t, cfg.PlacementHostAddr)
		assert.Equal(t, "", *cfg.PlacementHostAddr)
	})

	t.Run("default port appended when hostname provided without port", func(t *testing.T) {
		cfg := &RunConfig{SharedRunConfig: SharedRunConfig{PlacementHostAddr: strPtr("localhost")}}
		err := cfg.validatePlacementHostAddr()
		assert.NoError(t, err)
		if runtime.GOOS == daprWindowsOS {
			assert.True(t, strings.HasSuffix(*cfg.PlacementHostAddr, ":6050"))
		} else {
			assert.True(t, strings.HasSuffix(*cfg.PlacementHostAddr, ":50005"))
		}
	})

	t.Run("custom port preserved when provided", func(t *testing.T) {
		cfg := &RunConfig{SharedRunConfig: SharedRunConfig{PlacementHostAddr: strPtr("1.2.3.4:12345")}}
		err := cfg.validatePlacementHostAddr()
		assert.NoError(t, err)
		assert.Equal(t, "1.2.3.4:12345", *cfg.PlacementHostAddr)
	})
}

func TestValidateSchedulerHostAddr(t *testing.T) {
	t.Run("empty disables scheduler", func(t *testing.T) {
		cfg := &RunConfig{SharedRunConfig: SharedRunConfig{SchedulerHostAddress: strPtr("")}}
		err := cfg.validateSchedulerHostAddr()
		assert.NoError(t, err)
		assert.NotNil(t, cfg.SchedulerHostAddress)
		assert.Equal(t, "", *cfg.SchedulerHostAddress)
	})

	t.Run("whitespace disables scheduler", func(t *testing.T) {
		cfg := &RunConfig{SharedRunConfig: SharedRunConfig{SchedulerHostAddress: strPtr("  ")}}
		err := cfg.validateSchedulerHostAddr()
		assert.NoError(t, err)
		assert.NotNil(t, cfg.SchedulerHostAddress)
		assert.Equal(t, "", *cfg.SchedulerHostAddress)
	})

	t.Run("default port appended when hostname provided without port", func(t *testing.T) {
		cfg := &RunConfig{SharedRunConfig: SharedRunConfig{SchedulerHostAddress: strPtr("localhost")}}
		err := cfg.validateSchedulerHostAddr()
		assert.NoError(t, err)
		if runtime.GOOS == daprWindowsOS {
			assert.True(t, strings.HasSuffix(*cfg.SchedulerHostAddress, ":6060"))
		} else {
			assert.True(t, strings.HasSuffix(*cfg.SchedulerHostAddress, ":50006"))
		}
	})

	t.Run("custom port preserved when provided", func(t *testing.T) {
		cfg := &RunConfig{SharedRunConfig: SharedRunConfig{SchedulerHostAddress: strPtr("1.2.3.4:45678")}}
		err := cfg.validateSchedulerHostAddr()
		assert.NoError(t, err)
		assert.Equal(t, "1.2.3.4:45678", *cfg.SchedulerHostAddress)
	})
}
