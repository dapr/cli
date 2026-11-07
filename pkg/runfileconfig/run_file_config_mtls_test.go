/*
Copyright 2026 The Dapr Authors
Licensed under the Apache License, Version 2.0
(the "License");
you may not use this file except in compliance with
the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed
on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS
OF ANY KIND, either express or implied.
See the License for the specific language governing
permissions and limitations under the License.
*/

package runfileconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dapr/cli/pkg/standalone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func setupRunFileMTLSTest(
	t *testing.T,
	mtlsEnabled bool,
	createCerts bool,
) (string, string) {
	t.Helper()

	runtimePath := t.TempDir()
	t.Setenv("DAPR_RUNTIME_PATH", runtimePath)

	daprDir, err := standalone.GetDaprRuntimePath(runtimePath)
	require.NoError(t, err)

	require.NoError(t, os.MkdirAll(daprDir, 0o755))

	config := fmt.Sprintf(`apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec:
  mtls:
    enabled: %t
`, mtlsEnabled)

	configFile := standalone.GetDaprConfigPath(daprDir)

	require.NoError(t, os.WriteFile(
		configFile,
		[]byte(config),
		0o600,
	))

	if createCerts {
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

	return daprDir, configFile
}

func createMTLSRunFile(
	t *testing.T,
	configFile string,
	appDir string,
	appEnv map[string]string,
) string {
	t.Helper()

	config := RunFileConfig{
		Version: 1,
		Common: Common{
			SharedRunConfig: standalone.SharedRunConfig{
				ConfigFile: configFile,
			},
		},
		Apps: []App{
			{
				AppDirPath: appDir,
				RunConfig: standalone.RunConfig{
					SharedRunConfig: standalone.SharedRunConfig{
						Env: appEnv,
					},
					AppID: "mtls-test-app",
				},
			},
		},
	}

	content, err := yaml.Marshal(&config)
	require.NoError(t, err)

	runFile := filepath.Join(t.TempDir(), "dapr.yaml")
	require.NoError(t, os.WriteFile(runFile, content, 0o600))

	return runFile
}

func TestGetAppsMTLSEnv(t *testing.T) {
	t.Run("loads generated certificates and preserves app environment", func(t *testing.T) {
		_, configFile := setupRunFileMTLSTest(t, true, true)

		appDir := t.TempDir()
		runFile := createMTLSRunFile(
			t,
			configFile,
			appDir,
			map[string]string{
				"DAPR_CERT_KEY": "user-defined-key",
				"CUSTOM_ENV":    "application-value",
			},
		)

		cfg := &RunFileConfig{}

		apps, err := cfg.GetApps(runFile)

		require.NoError(t, err)
		require.Len(t, apps, 1)

		assert.Equal(t, "CA", apps[0].Env[standalone.EnvDaprTrustAnchors])
		assert.Equal(t, "CHAIN", apps[0].Env[standalone.EnvDaprCertChain])

		// Application-defined environment takes precedence.
		assert.Equal(t, "user-defined-key", apps[0].Env[standalone.EnvDaprCertKey])
		assert.Equal(t, "application-value", apps[0].Env["CUSTOM_ENV"])
	})

	t.Run("disabled mTLS does not inject existing certificates", func(t *testing.T) {
		_, configFile := setupRunFileMTLSTest(t, false, true)

		appDir := t.TempDir()
		runFile := createMTLSRunFile(t, configFile, appDir, nil)

		cfg := &RunFileConfig{}

		apps, err := cfg.GetApps(runFile)

		require.NoError(t, err)
		require.Len(t, apps, 1)

		assert.NotContains(t, apps[0].Env, standalone.EnvDaprTrustAnchors)
		assert.NotContains(t, apps[0].Env, standalone.EnvDaprCertChain)
		assert.NotContains(t, apps[0].Env, standalone.EnvDaprCertKey)
	})

	t.Run("missing certificates return an error when mTLS is enabled", func(t *testing.T) {
		_, configFile := setupRunFileMTLSTest(t, true, false)

		appDir := t.TempDir()
		runFile := createMTLSRunFile(t, configFile, appDir, nil)

		cfg := &RunFileConfig{}

		_, err := cfg.GetApps(runFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "sentry certs")
	})
}
