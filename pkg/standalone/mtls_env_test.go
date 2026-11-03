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

const (
	testMTLSEnabledConfig = `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec:
  mtls:
    enabled: true
`

	testMTLSDisabledConfig = `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec:
  mtls:
    enabled: false
`
)

func writeMTLSTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	require.NoError(t, os.MkdirAll(dir, 0o755))

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func writeMTLSTestCerts(t *testing.T, daprDir string) string {
	t.Helper()

	certsDir := GetDaprSentryCertsPath(daprDir)
	require.NoError(t, os.MkdirAll(certsDir, 0o755))

	for _, cert := range mtlsCertFiles {
		require.NoError(t, os.WriteFile(
			filepath.Join(certsDir, cert.file),
			[]byte(" certificate content \n"),
			0o600,
		))
	}

	return certsDir
}

func TestIsMTLSEnabled(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		want    bool
	}{
		{
			name:   "enabled",
			config: testMTLSEnabledConfig,
			want:   true,
		},
		{
			name:   "disabled",
			config: testMTLSDisabledConfig,
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
			configFile := writeMTLSTestFile(
				t,
				t.TempDir(),
				"config.yaml",
				tc.config,
			)

			assert.Equal(t, tc.want, IsMTLSEnabled(configFile))
		})
	}

	t.Run("missing config file", func(t *testing.T) {
		configFile := filepath.Join(t.TempDir(), "missing.yaml")

		assert.False(t, IsMTLSEnabled(configFile))
	})
}

func TestEnsureCerts(t *testing.T) {
	t.Run("all certificates exist", func(t *testing.T) {
		certsDir := writeMTLSTestCerts(t, t.TempDir())

		assert.True(t, EnsureCerts(certsDir))
	})

	t.Run("certificate directory does not exist", func(t *testing.T) {
		certsDir := filepath.Join(t.TempDir(), "certs")

		assert.False(t, EnsureCerts(certsDir))
	})

	t.Run("certificate is missing", func(t *testing.T) {
		certsDir := writeMTLSTestCerts(t, t.TempDir())

		require.NoError(t, os.Remove(filepath.Join(certsDir, "issuer.key")))

		assert.False(t, EnsureCerts(certsDir))
	})
}

func TestMTLSEnv(t *testing.T) {
	t.Run("loads all certificate environment variables", func(t *testing.T) {
		daprDir := t.TempDir()
		writeMTLSTestCerts(t, daprDir)

		env, err := MTLSEnv(daprDir)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			EnvDaprTrustAnchors: "certificate content",
			EnvDaprCertChain:    "certificate content",
			EnvDaprCertKey:      "certificate content",
		}, env)
	})

	t.Run("missing certificates are omitted", func(t *testing.T) {
		daprDir := t.TempDir()
		certsDir := writeMTLSTestCerts(t, daprDir)

		require.NoError(t, os.Remove(filepath.Join(certsDir, "issuer.key")))

		env, err := MTLSEnv(daprDir)

		require.NoError(t, err)
		assert.NotContains(t, env, EnvDaprCertKey)
		assert.Contains(t, env, EnvDaprTrustAnchors)
		assert.Contains(t, env, EnvDaprCertChain)
	})

	t.Run("missing certificate directory returns empty environment", func(t *testing.T) {
		env, err := MTLSEnv(t.TempDir())

		require.NoError(t, err)
		assert.Empty(t, env)
	})

	t.Run("unreadable certificate returns an error", func(t *testing.T) {
		daprDir := t.TempDir()
		certsDir := writeMTLSTestCerts(t, daprDir)

		keyPath := filepath.Join(certsDir, "issuer.key")

		require.NoError(t, os.Remove(keyPath))
		require.NoError(t, os.Mkdir(keyPath, 0o700))

		env, err := MTLSEnv(daprDir)

		require.Error(t, err)
		assert.Nil(t, env)
	})
}
