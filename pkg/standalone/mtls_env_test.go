
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
	mtlsEnabledConfig = `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec:
  mtls:
    enabled: true
`

	mtlsDisabledConfig = `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec:
  mtls:
    enabled: false
`
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	require.NoError(t, os.MkdirAll(dir, 0o755))

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func writeTestCerts(t *testing.T, daprDir string) string {
	t.Helper()

	certsDir := GetDaprSentryCertsPath(daprDir)
	require.NoError(t, os.MkdirAll(certsDir, 0o755))

	for _, cert := range mtlsCertFiles {
		require.NoError(t, os.WriteFile(
			filepath.Join(certsDir, cert.file),
			[]byte("certificate content"),
			0o600,
		))
	}

	return certsDir
}

func TestGetDaprSentryCertsPath(t *testing.T) {
	assert.Equal(
		t,
		filepath.Join("dapr", "certs"),
		GetDaprSentryCertsPath("dapr"),
	)
}

func TestGetMTLSConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
		wantErr bool
	}{
		{
			name:    "enabled",
			content: mtlsEnabledConfig,
			want:    true,
		},
		{
			name:    "disabled",
			content: mtlsDisabledConfig,
			want:    false,
		},
		{
			name:    "mtls omitted",
			content: "spec: {}\n",
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTestFile(t, t.TempDir(), "config.yaml", tc.content)

			cfg, err := GetMTLSConfig(path)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, cfg.Spec.MTLS.Enabled)
		})
	}

	t.Run("missing file", func(t *testing.T) {
		_, err := GetMTLSConfig(filepath.Join(t.TempDir(), "missing.yaml"))
		require.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestIsMTLSEnabled(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{
			name: "enabled",
			data: mtlsEnabledConfig,
			want: true,
		},
		{
			name: "disabled",
			data: mtlsDisabledConfig,
		},
		{
			name: "omitted",
			data: "spec: {}\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTestFile(t, t.TempDir(), "config.yaml", tc.data)
			assert.Equal(t, tc.want, IsMTLSEnabled(path))
		})
	}

	t.Run("invalid configuration", func(t *testing.T) {
		path := writeTestFile(t, t.TempDir(), "config.yaml", "spec: [invalid")
		assert.False(t, IsMTLSEnabled(path))
	})
}

func TestEnsureCerts(t *testing.T) {
	t.Run("all certificates exist", func(t *testing.T) {
		certsDir := writeTestCerts(t, t.TempDir())
		assert.True(t, EnsureCerts(certsDir))
	})

	t.Run("certificate directory does not exist", func(t *testing.T) {
		assert.False(t, EnsureCerts(filepath.Join(t.TempDir(), "certs")))
	})

	for _, missing := range mtlsCertFiles {
		t.Run("missing "+missing.file, func(t *testing.T) {
			certsDir := writeTestCerts(t, t.TempDir())

			require.NoError(t, os.Remove(filepath.Join(certsDir, missing.file)))

			assert.False(t, EnsureCerts(certsDir))
		})
	}

	t.Run("certificate path is a directory", func(t *testing.T) {
		certsDir := writeTestCerts(t, t.TempDir())

		path := filepath.Join(certsDir, "issuer.key")
		require.NoError(t, os.Remove(path))
		require.NoError(t, os.Mkdir(path, 0o700))

		// A directory is not a valid certificate file.
		assert.False(t, EnsureCerts(certsDir))
	})
}

func TestMTLSEnv(t *testing.T) {
	t.Run("all certificates are loaded", func(t *testing.T) {
		daprDir := t.TempDir()
		certsDir := writeTestCerts(t, daprDir)

		env, err := MTLSEnv(daprDir)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			EnvDaprTrustAnchors: "certificate content",
			EnvDaprCertChain:    "certificate content",
			EnvDaprCertKey:      "certificate content",
		}, env)

		assert.FileExists(t, filepath.Join(certsDir, "ca.crt"))
	})

	t.Run("missing certificates are omitted", func(t *testing.T) {
		daprDir := t.TempDir()
		certsDir := writeTestCerts(t, daprDir)

		require.NoError(t, os.Remove(filepath.Join(certsDir, "issuer.key")))

		env, err := MTLSEnv(daprDir)

		require.NoError(t, err)
		assert.NotContains(t, env, EnvDaprCertKey)
		assert.Contains(t, env, EnvDaprTrustAnchors)
		assert.Contains(t, env, EnvDaprCertChain)
	})

	t.Run("missing certificate directory", func(t *testing.T) {
		env, err := MTLSEnv(t.TempDir())

		require.NoError(t, err)
		assert.Empty(t, env)
	})

	t.Run("unreadable certificate returns an error", func(t *testing.T) {
		daprDir := t.TempDir()
		certsDir := writeTestCerts(t, daprDir)

		path := filepath.Join(certsDir, "issuer.key")
		require.NoError(t, os.Remove(path))
		require.NoError(t, os.Mkdir(path, 0o700))

		env, err := MTLSEnv(daprDir)

		require.Error(t, err)
		assert.Nil(t, env)
		assert.Contains(t, err.Error(), "reading mTLS file")
	})
}
