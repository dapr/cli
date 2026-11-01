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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapr/cli/pkg/standalone"
)

// These tests describe how the run file handles mTLS today. Where the current
// behavior looks questionable, the test says so in a comment.

// useDaprRuntimeDir points the Dapr runtime path at a temporary directory and
// returns the resulting .dapr directory.
func useDaprRuntimeDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("DAPR_RUNTIME_PATH", root)
	return filepath.Join(root, standalone.DefaultDaprDirName)
}

func writeSentryCerts(t *testing.T, daprDir string) {
	t.Helper()
	certsDir := standalone.GetDaprSentryCertsPath(daprDir)
	require.NoError(t, os.MkdirAll(certsDir, 0o755))
	for name, content := range map[string]string{
		"ca.crt":     "CA\n",
		"issuer.crt": "CHAIN\n",
		"issuer.key": "KEY\n",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(certsDir, name), []byte(content), 0o600))
	}
}

func commonWithEnv(env map[string]string) Common {
	return Common{SharedRunConfig: standalone.SharedRunConfig{Env: env}}
}

func appWithEnv(env map[string]string) App {
	return App{RunConfig: standalone.RunConfig{SharedRunConfig: standalone.SharedRunConfig{Env: env}}}
}

func TestAddMTLSEnvToCommon(t *testing.T) {
	t.Run("certs present adds the three variables to the common env", func(t *testing.T) {
		writeSentryCerts(t, useDaprRuntimeDir(t))
		cfg := &RunFileConfig{}

		require.NoError(t, cfg.addMTLSEnvToCommon())

		assert.Equal(t, map[string]string{
			standalone.EnvDaprTrustAnchors: "CA",
			standalone.EnvDaprCertChain:    "CHAIN",
			standalone.EnvDaprCertKey:      "KEY",
		}, cfg.Common.Env)
	})

	t.Run("values already in the common env win", func(t *testing.T) {
		writeSentryCerts(t, useDaprRuntimeDir(t))
		cfg := &RunFileConfig{Common: commonWithEnv(map[string]string{
			standalone.EnvDaprCertKey: "mine",
			"OTHER":                   "kept",
		})}

		require.NoError(t, cfg.addMTLSEnvToCommon())

		assert.Equal(t, "mine", cfg.Common.Env[standalone.EnvDaprCertKey])
		assert.Equal(t, "kept", cfg.Common.Env["OTHER"])
		assert.Equal(t, "CA", cfg.Common.Env[standalone.EnvDaprTrustAnchors])
		assert.Equal(t, "CHAIN", cfg.Common.Env[standalone.EnvDaprCertChain])
	})

	t.Run("no certs is an error", func(t *testing.T) {
		// Nothing here checks whether mTLS is enabled, so a machine that never
		// ran `dapr init --enable-mtls` gets this error for every run file.
		useDaprRuntimeDir(t)
		cfg := &RunFileConfig{}

		err := cfg.addMTLSEnvToCommon()

		assert.EqualError(t, err, "error getting sentry certs")
		assert.Empty(t, cfg.Common.Env)
	})

	t.Run("an mtls-disabled configuration does not skip the cert check", func(t *testing.T) {
		daprDir := useDaprRuntimeDir(t)
		require.NoError(t, os.MkdirAll(daprDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(daprDir, standalone.DefaultConfigFileName),
			[]byte("spec:\n  mtls:\n    enabled: false\n"), 0o600))
		cfg := &RunFileConfig{Common: Common{SharedRunConfig: standalone.SharedRunConfig{
			ConfigFile: filepath.Join(daprDir, standalone.DefaultConfigFileName),
		}}}

		assert.Error(t, cfg.addMTLSEnvToCommon())
	})
}

func TestMergeCommonAndAppsEnv(t *testing.T) {
	t.Run("an app without env gets a copy of the common env", func(t *testing.T) {
		cfg := &RunFileConfig{
			Common: commonWithEnv(map[string]string{"A": "1", "B": "2"}),
			Apps:   []App{appWithEnv(nil)},
		}

		cfg.mergeCommonAndAppsEnv()

		assert.Equal(t, map[string]string{"A": "1", "B": "2"}, cfg.Apps[0].Env)
		// It is a copy: changing the app's env leaves the common env alone.
		cfg.Apps[0].Env["A"] = "changed"
		assert.Equal(t, "1", cfg.Common.Env["A"])
	})

	t.Run("app values win over common values", func(t *testing.T) {
		cfg := &RunFileConfig{
			Common: commonWithEnv(map[string]string{"A": "common", "B": "common"}),
			Apps:   []App{appWithEnv(map[string]string{"A": "app"})},
		}

		cfg.mergeCommonAndAppsEnv()

		assert.Equal(t, map[string]string{"A": "app", "B": "common"}, cfg.Apps[0].Env)
	})

	t.Run("every app is merged independently", func(t *testing.T) {
		cfg := &RunFileConfig{
			Common: commonWithEnv(map[string]string{"A": "common"}),
			Apps:   []App{appWithEnv(nil), appWithEnv(map[string]string{"A": "second"})},
		}

		cfg.mergeCommonAndAppsEnv()

		assert.Equal(t, "common", cfg.Apps[0].Env["A"])
		assert.Equal(t, "second", cfg.Apps[1].Env["A"])
	})

	t.Run("no common env leaves apps untouched", func(t *testing.T) {
		cfg := &RunFileConfig{Apps: []App{appWithEnv(nil), appWithEnv(map[string]string{"A": "1"})}}

		cfg.mergeCommonAndAppsEnv()

		assert.Nil(t, cfg.Apps[0].Env)
		assert.Equal(t, map[string]string{"A": "1"}, cfg.Apps[1].Env)
	})
}

func TestGetAppsMTLSEnvReachesEveryApp(t *testing.T) {
	daprDir := useDaprRuntimeDir(t)
	writeSentryCerts(t, daprDir)

	appDir := t.TempDir()
	runFile := filepath.Join(t.TempDir(), "dapr.yaml")
	require.NoError(t, os.WriteFile(runFile, []byte("version: 1\napps:\n  - appDirPath: "+appDir+"\n"), 0o600))

	cfg := &RunFileConfig{}
	apps, err := cfg.GetApps(runFile)

	require.NoError(t, err)
	require.Len(t, apps, 1)
	// The variables land in the app's shared Env, which is passed to the
	// application process as well as to daprd.
	assert.Equal(t, "KEY", apps[0].Env[standalone.EnvDaprCertKey])
	assert.Equal(t, "CA", apps[0].Env[standalone.EnvDaprTrustAnchors])
	assert.Equal(t, "CHAIN", apps[0].Env[standalone.EnvDaprCertChain])
}

func TestGetAppsFailsWithoutSentryCerts(t *testing.T) {
	useDaprRuntimeDir(t)

	appDir := t.TempDir()
	runFile := filepath.Join(t.TempDir(), "dapr.yaml")
	require.NoError(t, os.WriteFile(runFile, []byte("version: 1\napps:\n  - appDirPath: "+appDir+"\n"), 0o600))

	cfg := &RunFileConfig{}
	apps, err := cfg.GetApps(runFile)

	// Even a run file that never mentions mTLS can't be loaded without certs.
	assert.Nil(t, apps)
	assert.EqualError(t, err, "error getting sentry certs")
}
