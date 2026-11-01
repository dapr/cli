
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

package runfileconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapr/cli/pkg/standalone"
)

func setupMTLSTest(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	t.Setenv("DAPR_RUNTIME_PATH", root)

	return filepath.Join(root, standalone.DefaultDaprDirName)
}

func createMTLSTestCerts(t *testing.T, daprDir string) {
	t.Helper()

	certsDir := standalone.GetDaprSentryCertsPath(daprDir)
	require.NoError(t, os.MkdirAll(certsDir, 0o755))

	for _, cert := range []struct {
		name    string
		content string
	}{
		{"ca.crt", "CA"},
		{"issuer.crt", "CHAIN"},
		{"issuer.key", "KEY"},
	} {
		require.NoError(t, os.WriteFile(
			filepath.Join(certsDir, cert.name),
			[]byte(cert.content),
			0o600,
		))
	}
}

func createRunFile(t *testing.T, appDir string) string {
	t.Helper()

	runFile := filepath.Join(t.TempDir(), "dapr.yaml")
	content := fmt.Sprintf(`version: 1
apps:
  - appDirPath: %q
`, appDir)

	require.NoError(t, os.WriteFile(runFile, []byte(content), 0o600))

	return runFile
}

func commonWithEnv(env map[string]string) Common {
	return Common{
		SharedRunConfig: standalone.SharedRunConfig{
			Env: env,
		},
	}
}

func appWithEnv(env map[string]string) App {
	return App{
		RunConfig: standalone.RunConfig{
			SharedRunConfig: standalone.SharedRunConfig{
				Env: env,
			},
		},
	}
}

func TestAddMTLSEnvToCommon(t *testing.T) {
	t.Run("mTLS disabled does not require certificates", func(t *testing.T) {
		daprDir := setupMTLSTest(t)
		require.NoError(t, os.MkdirAll(daprDir, 0o755))

		configFile := standalone.GetDaprConfigPath(daprDir)
		require.NoError(t, os.WriteFile(
			configFile,
			[]byte("spec:\n  mtls:\n    enabled: false\n"),
			0o600,
		))

		cfg := &RunFileConfig{
			Common: Common{
				SharedRunConfig: standalone.SharedRunConfig{
					ConfigFile: configFile,
				},
			},
		}

		require.NoError(t, cfg.addMTLSEnvToCommon())
		assert.Empty(t, cfg.Common.Env)
	})

	t.Run("mTLS enabled loads certificate variables", func(t *testing.T) {
		daprDir := setupMTLSTest(t)
		createMTLSTestCerts(t, daprDir)

		configFile := standalone.GetDaprConfigPath(daprDir)
		require.NoError(t, os.WriteFile(
			configFile,
			[]byte("spec:\n  mtls:\n    enabled: true\n"),
			0o600,
		))

		cfg := &RunFileConfig{
			Common: Common{
				SharedRunConfig: standalone.SharedRunConfig{
					ConfigFile: configFile,
				},
			},
		}

		require.NoError(t, cfg.addMTLSEnvToCommon())

		assert.Equal(t, map[string]string{
			standalone.EnvDaprTrustAnchors: "CA",
			standalone.EnvDaprCertChain:    "CHAIN",
			standalone.EnvDaprCertKey:      "KEY",
		}, cfg.Common.Env)
	})

	t.Run("user-defined values take precedence", func(t *testing.T) {
		daprDir := setupMTLSTest(t)
		createMTLSTestCerts(t, daprDir)

		configFile := standalone.GetDaprConfigPath(daprDir)
		require.NoError(t, os.WriteFile(
			configFile,
			[]byte("spec:\n  mtls:\n    enabled: true\n"),
			0o600,
		))

		cfg := &RunFileConfig{
			Common: Common{
				SharedRunConfig: standalone.SharedRunConfig{
					ConfigFile: configFile,
					Env: map[string]string{
						standalone.EnvDaprCertKey: "custom-key",
						"OTHER":                   "preserved",
					},
				},
			},
		}

		require.NoError(t, cfg.addMTLSEnvToCommon())

		assert.Equal(t, "custom-key", cfg.Common.Env[standalone.EnvDaprCertKey])
		assert.Equal(t, "CA", cfg.Common.Env[standalone.EnvDaprTrustAnchors])
		assert.Equal(t, "CHAIN", cfg.Common.Env[standalone.EnvDaprCertChain])
		assert.Equal(t, "preserved", cfg.Common.Env["OTHER"])
	})

	t.Run("missing certificates return an error when mTLS is enabled", func(t *testing.T) {
		daprDir := setupMTLSTest(t)
		require.NoError(t, os.MkdirAll(daprDir, 0o755))

		configFile := standalone.GetDaprConfigPath(daprDir)
		require.NoError(t, os.WriteFile(
			configFile,
			[]byte("spec:\n  mtls:\n    enabled: true\n"),
			0o600,
		))

		cfg := &RunFileConfig{
			Common: Common{
				SharedRunConfig: standalone.SharedRunConfig{
					ConfigFile: configFile,
				},
			},
		}

		err := cfg.addMTLSEnvToCommon()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sentry certs")
		assert.Empty(t, cfg.Common.Env)
	})
}

func TestMergeCommonAndAppsEnv(t *testing.T) {
	tests := []struct {
		name       string
		commonEnv  map[string]string
		appEnv     map[string]string
		want       map[string]string
		wantNilEnv bool
	}{
		{
			name:      "common environment is copied",
			commonEnv: map[string]string{"A": "1", "B": "2"},
			want:      map[string]string{"A": "1", "B": "2"},
		},
		{
			name:      "app environment takes precedence",
			commonEnv: map[string]string{"A": "common", "B": "common"},
			appEnv:    map[string]string{"A": "app"},
			want:      map[string]string{"A": "app", "B": "common"},
		},
		{
			name:       "empty common environment leaves nil app environment",
			wantNilEnv: true,
		},
		{
			name:      "empty app environment is preserved when common is nil",
			appEnv:    map[string]string{"A": "app"},
			want:      map[string]string{"A": "app"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &RunFileConfig{
				Common: commonWithEnv(tc.commonEnv),
				Apps:   []App{appWithEnv(tc.appEnv)},
			}

			cfg.mergeCommonAndAppsEnv()

			if tc.wantNilEnv {
				assert.Nil(t, cfg.Apps[0].Env)
				return
			}

			assert.Equal(t, tc.want, cfg.Apps[0].Env)

			if len(tc.commonEnv) > 0 {
				cfg.Apps[0].Env["A"] = "modified"
				assert.Equal(t, tc.commonEnv["A"], cfg.Common.Env["A"])
			}
		})
	}
}

func TestGetAppsMTLSEnv(t *testing.T) {
	daprDir := setupMTLSTest(t)
	createMTLSTestCerts(t, daprDir)

	configFile := standalone.GetDaprConfigPath(daprDir)
	require.NoError(t, os.WriteFile(
		configFile,
		[]byte("spec:\n  mtls:\n    enabled: true\n"),
		0o600,
	))

	appDir := t.TempDir()
	runFile := createRunFile(t, appDir)

	cfg := &RunFileConfig{
		Common: Common{
			SharedRunConfig: standalone.SharedRunConfig{
				ConfigFile: configFile,
			},
		},
	}

	apps, err := cfg.GetApps(runFile)

	require.NoError(t, err)
	require.Len(t, apps, 1)

	assert.Equal(t, "CA", apps[0].Env[standalone.EnvDaprTrustAnchors])
	assert.Equal(t, "CHAIN", apps[0].Env[standalone.EnvDaprCertChain])
	assert.Equal(t, "KEY", apps[0].Env[standalone.EnvDaprCertKey])
}

func TestGetAppsWithoutMTLS(t *testing.T) {
	setupMTLSTest(t)

	appDir := t.TempDir()
	runFile := createRunFile(t, appDir)

	cfg := &RunFileConfig{}

	apps, err := cfg.GetApps(runFile)

	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.NotContains(t, apps[0].Env, standalone.EnvDaprCertKey)
}
