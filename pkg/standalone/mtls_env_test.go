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

// These tests describe how the self-hosted mTLS code behaves today. Where the
// current behavior looks questionable, the test says so in a comment.

const (
	testMTLSOnConfig  = "apiVersion: dapr.io/v1alpha1\nkind: Configuration\nspec:\n  mtls:\n    enabled: true\n"
	testMTLSOffConfig = "apiVersion: dapr.io/v1alpha1\nkind: Configuration\nspec:\n  mtls:\n    enabled: false\n"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

// writeTestCerts writes the given certificate files into daprDir's certs
// directory and returns that directory.
func writeTestCerts(t *testing.T, daprDir string, names ...string) string {
	t.Helper()
	certsDir := GetDaprSentryCertsPath(daprDir)
	for _, n := range names {
		writeTestFile(t, certsDir, n, "  value of "+n+"\n\n")
	}
	return certsDir
}

func TestGetDaprSentryCertsPath(t *testing.T) {
	assert.Equal(t, filepath.Join("some", "dir", "certs"), GetDaprSentryCertsPath(filepath.Join("some", "dir")))
}

func TestGetMTLSConfig(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing file returns the read error", func(t *testing.T) {
		_, err := GetMTLSConfig(filepath.Join(dir, "missing.yaml"))
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("invalid yaml returns an error", func(t *testing.T) {
		_, err := GetMTLSConfig(writeTestFile(t, dir, "bad.yaml", "spec: [unclosed"))
		assert.Error(t, err)
	})

	t.Run("mtls enabled", func(t *testing.T) {
		cfg, err := GetMTLSConfig(writeTestFile(t, dir, "on.yaml", testMTLSOnConfig))
		require.NoError(t, err)
		assert.True(t, cfg.Spec.MTLS.Enabled)
	})

	t.Run("mtls disabled", func(t *testing.T) {
		cfg, err := GetMTLSConfig(writeTestFile(t, dir, "off.yaml", testMTLSOffConfig))
		require.NoError(t, err)
		assert.False(t, cfg.Spec.MTLS.Enabled)
	})

	t.Run("mtls not mentioned", func(t *testing.T) {
		cfg, err := GetMTLSConfig(writeTestFile(t, dir, "none.yaml", "spec: {}\n"))
		require.NoError(t, err)
		assert.False(t, cfg.Spec.MTLS.Enabled)
	})
}

func TestIsMTLSEnabled(t *testing.T) {
	dir := t.TempDir()

	assert.True(t, IsMTLSEnabled(writeTestFile(t, dir, "on.yaml", testMTLSOnConfig)))
	assert.False(t, IsMTLSEnabled(writeTestFile(t, dir, "off.yaml", testMTLSOffConfig)))

	// Every failure is reported as "not enabled"; the cause is dropped.
	assert.False(t, IsMTLSEnabled(""))
	assert.False(t, IsMTLSEnabled(filepath.Join(dir, "missing.yaml")))
	assert.False(t, IsMTLSEnabled(writeTestFile(t, dir, "bad.yaml", "spec: [unclosed")))
	assert.False(t, IsMTLSEnabled(dir))
}

func TestEnsureCerts(t *testing.T) {
	t.Run("all three files present", func(t *testing.T) {
		daprDir := t.TempDir()
		certsDir := writeTestCerts(t, daprDir, "ca.crt", "issuer.crt", "issuer.key")
		assert.True(t, EnsureCerts(certsDir))
	})

	t.Run("certs directory missing", func(t *testing.T) {
		assert.False(t, EnsureCerts(filepath.Join(t.TempDir(), "certs")))
	})

	for _, missing := range []string{"ca.crt", "issuer.crt", "issuer.key"} {
		t.Run("missing "+missing, func(t *testing.T) {
			var present []string
			for _, n := range []string{"ca.crt", "issuer.crt", "issuer.key"} {
				if n != missing {
					present = append(present, n)
				}
			}
			certsDir := writeTestCerts(t, t.TempDir(), present...)
			assert.False(t, EnsureCerts(certsDir))
		})
	}

	t.Run("only a missing file counts as absent", func(t *testing.T) {
		// A file that exists but can't be read (here: a directory) is not
		// ErrNotExist, so EnsureCerts still reports success.
		daprDir := t.TempDir()
		certsDir := writeTestCerts(t, daprDir, "ca.crt", "issuer.crt")
		require.NoError(t, os.MkdirAll(filepath.Join(certsDir, "issuer.key"), 0o755))
		assert.True(t, EnsureCerts(certsDir))
	})
}

func TestMTLSEnv(t *testing.T) {
	t.Run("reads each file into its variable and trims whitespace", func(t *testing.T) {
		daprDir := t.TempDir()
		writeTestCerts(t, daprDir, "ca.crt", "issuer.crt", "issuer.key")

		env, err := MTLSEnv(daprDir)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			EnvDaprTrustAnchors: "value of ca.crt",
			EnvDaprCertChain:    "value of issuer.crt",
			EnvDaprCertKey:      "value of issuer.key",
		}, env)
	})

	t.Run("NAMESPACE is declared but never set", func(t *testing.T) {
		daprDir := t.TempDir()
		writeTestCerts(t, daprDir, "ca.crt", "issuer.crt", "issuer.key")

		env, err := MTLSEnv(daprDir)

		require.NoError(t, err)
		assert.NotContains(t, env, EnvNamespace)
		assert.Equal(t, "NAMESPACE", EnvNamespace)
		assert.Equal(t, "default", defaultNamespace)
	})

	t.Run("no certs directory", func(t *testing.T) {
		env, err := MTLSEnv(t.TempDir())

		assert.Nil(t, env)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	for _, missing := range []string{"ca.crt", "issuer.crt", "issuer.key"} {
		t.Run("error names the first missing file: "+missing, func(t *testing.T) {
			var present []string
			for _, n := range []string{"ca.crt", "issuer.crt", "issuer.key"} {
				if n != missing {
					present = append(present, n)
				}
			}
			daprDir := t.TempDir()
			writeTestCerts(t, daprDir, present...)

			env, err := MTLSEnv(daprDir)

			assert.Nil(t, env)
			require.ErrorIs(t, err, os.ErrNotExist)
			assert.Contains(t, err.Error(), missing)
		})
	}

	t.Run("unreadable file is a read error, not a missing-file error", func(t *testing.T) {
		daprDir := t.TempDir()
		certsDir := writeTestCerts(t, daprDir, "ca.crt", "issuer.crt")
		require.NoError(t, os.MkdirAll(filepath.Join(certsDir, "issuer.key"), 0o755))

		env, err := MTLSEnv(daprDir)

		assert.Nil(t, env)
		require.Error(t, err)
		assert.NotErrorIs(t, err, os.ErrNotExist)
		assert.Contains(t, err.Error(), "reading mTLS file")
	})

	t.Run("does not look at the configuration file", func(t *testing.T) {
		// MTLSEnv loads certs whether or not mTLS is enabled anywhere; callers
		// are expected to check IsMTLSEnabled first.
		daprDir := t.TempDir()
		writeTestFile(t, daprDir, DefaultConfigFileName, testMTLSOffConfig)
		writeTestCerts(t, daprDir, "ca.crt", "issuer.crt", "issuer.key")

		env, err := MTLSEnv(daprDir)

		require.NoError(t, err)
		assert.Len(t, env, 3)
	})
}

func TestMTLSEndpoint(t *testing.T) {
	dir := t.TempDir()

	assert.Equal(t, "", mtlsEndpoint(""))
	assert.Equal(t, "", mtlsEndpoint(filepath.Join(dir, "missing.yaml")))
	assert.Equal(t, "", mtlsEndpoint(writeTestFile(t, dir, "off.yaml", testMTLSOffConfig)))
	assert.Equal(t, sentryDefaultAddress, mtlsEndpoint(writeTestFile(t, dir, "on.yaml", testMTLSOnConfig)))
}

func TestSentryDefaultAddress(t *testing.T) {
	// The address daprd is pointed at is fixed, whatever port Sentry is
	// published on for the platform.
	assert.Equal(t, "localhost:50001", sentryDefaultAddress)
}

func TestGetArgsMTLS(t *testing.T) {
	dir := t.TempDir()

	t.Run("mtls enabled adds the flags and sentry address", func(t *testing.T) {
		config := &RunConfig{
			AppID:           "testapp",
			SharedRunConfig: SharedRunConfig{ConfigFile: writeTestFile(t, dir, "on.yaml", testMTLSOnConfig)},
		}

		args := config.getArgs()

		assert.Contains(t, args, "--enable-mtls")
		i := indexOf(args, "--sentry-address")
		require.GreaterOrEqual(t, i, 0)
		require.Less(t, i+1, len(args))
		assert.Equal(t, sentryDefaultAddress, args[i+1])
	})

	t.Run("mtls disabled adds nothing", func(t *testing.T) {
		config := &RunConfig{
			AppID:           "testapp",
			SharedRunConfig: SharedRunConfig{ConfigFile: writeTestFile(t, dir, "off.yaml", testMTLSOffConfig)},
		}

		args := config.getArgs()

		assert.NotContains(t, args, "--enable-mtls")
		assert.NotContains(t, args, "--sentry-address")
	})

	t.Run("no config file adds nothing", func(t *testing.T) {
		args := (&RunConfig{AppID: "testapp"}).getArgs()

		assert.NotContains(t, args, "--enable-mtls")
		assert.NotContains(t, args, "--sentry-address")
	})

	t.Run("unreadable config file adds nothing", func(t *testing.T) {
		config := &RunConfig{
			AppID:           "testapp",
			SharedRunConfig: SharedRunConfig{ConfigFile: filepath.Join(dir, "missing.yaml")},
		}

		assert.NotContains(t, config.getArgs(), "--enable-mtls")
	})
}

func indexOf(s []string, v string) int {
	for i := range s {
		if s[i] == v {
			return i
		}
	}
	return -1
}

func TestSharedEnvReachesDaprdAndTheApp(t *testing.T) {
	// SharedRunConfig.Env is applied to both processes. Anything put there,
	// such as the mTLS variables, is therefore visible to the application too.
	config := &RunConfig{
		AppID:   "testapp",
		Command: []string{"echo"},
		SharedRunConfig: SharedRunConfig{
			Env: map[string]string{EnvDaprCertKey: "secret"},
		},
	}

	daprd, err := GetDaprCommand(config)
	require.NoError(t, err)
	app := GetAppCommand(config)
	require.NotNil(t, app)

	assert.Contains(t, daprd.Env, EnvDaprCertKey+"=secret")
	assert.Contains(t, app.Env, EnvDaprCertKey+"=secret")
}

func TestCreateDefaultConfigurationMTLS(t *testing.T) {
	t.Run("mtls true writes mtls enabled", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "config.yaml")

		require.NoError(t, createDefaultConfiguration("", file, true))

		content, err := os.ReadFile(file)
		require.NoError(t, err)
		assert.Equal(t, `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec:
  mtls:
    enabled: true
`, string(content))
		assert.True(t, IsMTLSEnabled(file))
	})

	t.Run("mtls true keeps tracing settings", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "config.yaml")

		require.NoError(t, createDefaultConfiguration("zipkin_host", file, true))

		content, err := os.ReadFile(file)
		require.NoError(t, err)
		assert.Contains(t, string(content), "endpointAddress: http://zipkin_host:9411/api/v2/spans")
		assert.True(t, IsMTLSEnabled(file))
	})

	t.Run("mtls false leaves the section out", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "config.yaml")

		require.NoError(t, createDefaultConfiguration("", file, false))

		content, err := os.ReadFile(file)
		require.NoError(t, err)
		assert.NotContains(t, string(content), "mtls")
		assert.False(t, IsMTLSEnabled(file))
	})

	t.Run("an existing file is never overwritten", func(t *testing.T) {
		file := writeTestFile(t, t.TempDir(), "config.yaml", testMTLSOffConfig)

		require.NoError(t, createDefaultConfiguration("", file, true))

		content, err := os.ReadFile(file)
		require.NoError(t, err)
		assert.Equal(t, testMTLSOffConfig, string(content))
		assert.False(t, IsMTLSEnabled(file))
	})
}
