//go:build e2e && !template
// +build e2e,!template

/*
Copyright 2022 The Dapr Authors
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

package standalone_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dapr/cli/pkg/standalone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getAirgapDirFromEnv(t *testing.T) string {
	var airgapDir string
	var ok bool
	airgapEnvVar := "DAPR_E2E_INIT_AIRGAP_DIR"
	if airgapDir, ok = os.LookupEnv(airgapEnvVar); ok {
		airgapDir = filepath.Clean(airgapDir)
	} else {
		t.Fatalf("env var \"%s\" not set", airgapEnvVar)
	}
	return airgapDir
}

func getVersionsFromBundle(t *testing.T, airgapDir string) (string, string) {
	bundleDet := standalone.BundleDetails{}
	detailsFilePath := filepath.Join(airgapDir, standalone.BundleDetailsFileName)
	err := bundleDet.ReadAndParseDetails(detailsFilePath)
	require.NoError(t, err, "error parsing details file from bundle location: %w", err)

	runtimeVersion := *bundleDet.RuntimeVersion
	dashboardVersion := *bundleDet.DashboardVersion

	return runtimeVersion, dashboardVersion
}

func TestStandaloneAirgap(t *testing.T) {

	t.Run("init with --from-dir flag", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		airgapDir := getAirgapDirFromEnv(t)

		args := []string{
			"--from-dir", airgapDir,
		}
		output, err := cmdInit(args...)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		daprPath := filepath.Join(homeDir, ".dapr")
		require.DirExists(t, daprPath, "Directory %s does not exist", daprPath)

		daprRuntimeVersion, daprDashboardVersion := getVersionsFromBundle(t, airgapDir)
		verifyContainers(t, daprRuntimeVersion, true)
		verifyBinaries(t, daprPath, daprRuntimeVersion, daprDashboardVersion)
		verifyConfigs(t, daprPath, true)
	})

}
