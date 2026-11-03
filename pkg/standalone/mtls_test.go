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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mtlsTestFlagValue(args []string, flag string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}

	return ""
}

func TestRunConfigMTLSArgs(t *testing.T) {
	t.Run("mTLS enabled", func(t *testing.T) {
		configFile := writeMTLSTestFile(
			t,
			t.TempDir(),
			"config.yaml",
			testMTLSEnabledConfig,
		)

		config := &RunConfig{
			SharedRunConfig: SharedRunConfig{
				ConfigFile: configFile,
			},
		}

		args := config.getArgs()

		assert.Contains(t, args, "--enable-mtls")
		assert.Contains(t, args, "--sentry-address")
		assert.Equal(t, SentryAddress(), mtlsTestFlagValue(args, "--sentry-address"))
	})

	t.Run("mTLS disabled", func(t *testing.T) {
		configFile := writeMTLSTestFile(
			t,
			t.TempDir(),
			"config.yaml",
			testMTLSDisabledConfig,
		)

		config := &RunConfig{
			SharedRunConfig: SharedRunConfig{
				ConfigFile: configFile,
			},
		}

		args := config.getArgs()

		assert.NotContains(t, args, "--enable-mtls")
		assert.NotContains(t, args, "--sentry-address")
	})

	t.Run("mTLS omitted", func(t *testing.T) {
		configFile := writeMTLSTestFile(
			t,
			t.TempDir(),
			"config.yaml",
			"spec: {}\n",
		)

		config := &RunConfig{
			SharedRunConfig: SharedRunConfig{
				ConfigFile: configFile,
			},
		}

		args := config.getArgs()

		assert.NotContains(t, args, "--enable-mtls")
		assert.NotContains(t, args, "--sentry-address")
	})
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
			name:    "docker without custom network",
			runtime: "docker",
		},
		{
			name:        "docker with custom network",
			runtime:     "docker",
			info:        initInfo{dockerNetwork: "dapr-network"},
			wantNetwork: true,
		},
		{
			name:    "podman without custom network",
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

			assert.Contains(t, args, image)
			assert.Contains(t, args, "--trust-domain="+sentryTrustDomain)
			assert.Contains(t, args, "--issuer-credentials="+sentryContainerCertsDir)
			assert.Contains(t, args, "--config")
			assert.Contains(t, args, sentryContainerConfigFile)

			if tc.wantNetwork {
				assert.Contains(t, args, "--network")
				assert.Contains(t, args, "dapr-network")
				assert.Contains(t, args, "--network-alias")
				assert.NotContains(t, args, "-p")
			} else {
				assert.Contains(t, args, "-p")
				assert.Contains(t, args, fmt.Sprintf(
					"%d:50001",
					sentryHostPort(),
				))
				assert.Contains(t, args, fmt.Sprintf("%d:8080", sentryHealthPort))
				assert.Contains(t, args, fmt.Sprintf("%d:9090", sentryMetricPort))
			}
		})
	}
}

func TestSentryAddress(t *testing.T) {
	require.Equal(t, fmt.Sprintf("localhost:%d", sentryHostPort()), SentryAddress())
}
