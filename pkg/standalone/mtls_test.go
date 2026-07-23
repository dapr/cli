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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentryAddressForContainer(t *testing.T) {
	assert.Equal(t, "dapr_sentry:50001", sentryAddressForContainer("dapr-network"))
	assert.Equal(t, sentryAddressHostGateway, sentryAddressForContainer(""))
}

func TestMTLSControlPlaneServiceArgs(t *testing.T) {
	args := mtlsControlPlaneServiceArgs("")
	assert.Contains(t, args, "--tls-enabled")
	assert.Contains(t, args, "--trust-domain")
	assert.Contains(t, args, defaultTrustDomain)
	assert.Contains(t, args, "--trust-anchors-file")
	assert.Contains(t, args, trustAnchorsContainerPath)
	assert.Contains(t, args, "--sentry-address")
	assert.Contains(t, args, sentryAddressHostGateway)
}

func TestBuildSentryContainerRunArgs(t *testing.T) {
	installDir := t.TempDir()
	certsDir := GetDaprCertsPath(installDir)
	require.NoError(t, os.MkdirAll(certsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(certsDir, trustAnchorsFile), []byte("test"), 0o600))
	require.NoError(t, os.WriteFile(GetDaprConfigPath(installDir), []byte("apiVersion: dapr.io/v1alpha1"), 0o644))

	info := initInfo{
		installDir: installDir,
		enableMTLS: true,
	}
	args := buildSentryContainerRunArgs(info, "daprio/dapr:1.18.1")

	assert.Contains(t, args, "--mode")
	assert.Contains(t, args, sentryStandaloneMode)
	assert.Contains(t, args, "--config")
	assert.Contains(t, args, sentryConfigContainerPath)
	assert.Contains(t, args, "--issuer-credentials")
	assert.Contains(t, args, credentialsContainerPath)
	assert.Contains(t, args, "--trust-domain")
	assert.Contains(t, args, defaultTrustDomain)
	assert.Contains(t, args, "dapr_sentry")
}

func TestMergeMTLSIntoConfiguration(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	existing := `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec:
  tracing:
    samplingRate: "1"
    zipkin:
      endpointAddress: http://localhost:9411/api/v2/spans
`
	require.NoError(t, os.WriteFile(configPath, []byte(existing), 0o644))

	require.NoError(t, mergeMTLSIntoConfiguration(configPath))

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	text := string(content)
	assert.Contains(t, text, "mtls:")
	assert.Contains(t, text, "enabled: true")
	assert.Contains(t, text, "zipkin:")
}

func TestGenerateCertsForMTLSInternal(t *testing.T) {
	installDir := t.TempDir()
	info := initInfo{
		installDir: installDir,
		enableMTLS: true,
	}

	require.NoError(t, generateCertsForMTLSInternal(info))

	certsDir := GetDaprCertsPath(installDir)
	for _, file := range []string{trustAnchorsFile, issuerCertFile, issuerKeyFile} {
		path := filepath.Join(certsDir, file)
		assert.FileExists(t, path)
		stat, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), stat.Mode().Perm())
	}

	require.NoError(t, generateCertsForMTLSInternal(info))
}

func TestContainersToRemoveIncludesSentry(t *testing.T) {
	containers := containersToRemove(true, false, false)
	for _, c := range containers {
		if c.name == DaprSentryContainerName {
			assert.False(t, c.warnIfMissing)
			return
		}
	}
	t.Fatal("expected sentry container in removal list")
}
