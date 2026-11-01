
/*
Copyright 2026 The Dapr Authors
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
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMTLSEnabled(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		want    bool
		wantErr bool
	}{
		{
			name:   "enabled",
			config: mtlsEnabledConfig,
			want:   true,
		},
		{
			name:   "disabled",
			config: mtlsDisabledConfig,
		},
		{
			name:   "mtls omitted",
			config: "spec: {}\n",
		},
		{
			name:   "unrelated configuration fields",
			config: "spec:\n  tracing:\n    samplingRate: \"1\"\n  mtls:\n    enabled: true\n",
			want:   true,
		},
		{
			name:    "invalid yaml",
			config:  "spec: [invalid",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseMTLSConfig([]byte(tc.config))

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Spec.MTLS.Enabled)
		})
	}
}

func TestMTLSEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		want       string
	}{
		{
			name:   "enabled",
			config: mtlsEnabledConfig,
			want:   sentryDefaultAddress,
		},
		{
			name:   "disabled",
			config: mtlsDisabledConfig,
		},
		{
			name:   "mtls omitted",
			config: "spec: {}\n",
		},
		{
			name:   "invalid yaml",
			config: "spec: [invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(configFile, []byte(tc.config), 0o600))

			assert.Equal(t, tc.want, mtlsEndpoint(configFile))
		})
	}

	t.Run("empty config path", func(t *testing.T) {
		assert.Empty(t, mtlsEndpoint(""))
	})

	t.Run("missing config file", func(t *testing.T) {
		assert.Empty(t, mtlsEndpoint(filepath.Join(t.TempDir(), "missing.yaml")))
	})
}

func TestSentryAddress(t *testing.T) {
	port := sentryPort
	if runtime.GOOS == daprWindowsOS {
		port = sentryWindowsPort
	}

	assert.Equal(t, fmt.Sprintf("localhost:%d", port), SentryAddress())
}

func TestSentryContainerArgs(t *testing.T) {
	const (
		containerName = "dapr_sentry"
		image         = "ghcr.io/dapr/dapr:1.17.0"
		certsDir      = "/tmp/dapr/certs"
		configFile    = "/tmp/dapr/config.yaml"
	)

	tests := []struct {
		name        string
		info        initInfo
		runtime     string
		wantNetwork bool
	}{
		{
			name:    "docker without network",
			runtime: "docker",
		},
		{
			name:        "docker with network",
			runtime:     "docker",
			info:        initInfo{dockerNetwork: "dapr-network"},
			wantNetwork: true,
		},
		{
			name:    "podman without network",
			runtime: "podman",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := sentryContainerArgs(
				tc.info,
				tc.runtime,
				containerName,
				image,
				certsDir,
				configFile,
			)

			assert.Contains(t, args, "--name")
			assert.Contains(t, args, containerName)

			assert.Contains(t, args, "-v")
			assert.Contains(t, args, certsDir+":"+sentryContainerCertsDir)
			assert.Contains(t, args, configFile+":"+sentryContainerConfigFile+":ro")

			assert.Equal(t, []string{
				image,
				"--trust-domain=" + sentryTrustDomain,
				"--issuer-credentials=" + sentryContainerCertsDir,
				"--config",
				sentryContainerConfigFile,
			}, args[len(args)-5:])

			if tc.wantNetwork {
				assert.Contains(t, args, "--network")
				assert.Contains(t, args, "dapr-network")
				assert.Contains(t, args, "--network-alias")
				assert.NotContains(t, args, "-p")
			} else {
				assert.Contains(t, args, "-p")
				assert.Contains(t, args, fmt.Sprintf("%d:50001", sentryHostPort()))
				assert.Contains(t, args, fmt.Sprintf("%d:8080", sentryHealthPort))
				assert.Contains(t, args, fmt.Sprintf("%d:9090", sentryMetricPort))
			}
		})
	}
}
