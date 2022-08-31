//go:build e2e
// +build e2e

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
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dapr/cli/tests/e2e/common"
	"github.com/dapr/cli/tests/e2e/spawn"
	"github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestStandaloneInit(t *testing.T) {
	daprRuntimeVersion, daprDashboardVersion := common.GetVersionsFromEnv(t)

	t.Run("init with invalid private registry", func(t *testing.T) {
		if isSlimMode() {
			t.Skip("Skipping init with private registry test because of slim installation")
		}

		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		output, err := cmdInit(daprRuntimeVersion, "--image-registry", "smplregistry.io/owner")
		t.Log(output)
		require.Error(t, err, "init failed")
	})

	t.Run("init should error if both --from-dir and --image-registry are given", func(t *testing.T) {
		if isSlimMode() {
			t.Skip("Skipping init with --image-registry and --from-dir test because of slim installation")
		}

		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		output, err := cmdInit(daprRuntimeVersion, "--image-registry", "localhost:5000", "--from-dir", "./local-dir")
		require.Error(t, err, "expected error if both flags are given")
		require.Contains(t, output, "both --image-registry and --from-dir flags cannot be given at the same time")
	})

	t.Run("init should error out if container runtime is not valid", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		output, err := cmdInit(daprRuntimeVersion, "--container-runtime", "invalid")
		require.Error(t, err, "expected error if container runtime is invalid")
		require.Contains(t, output, "Invalid container runtime")
	})

	t.Run("init", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		output, err := cmdInit(daprRuntimeVersion)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		daprPath := filepath.Join(homeDir, ".dapr")
		require.DirExists(t, daprPath, "Directory %s does not exist", daprPath)

		verifyContainers(t, daprRuntimeVersion)
		verifyBinaries(t, daprPath, daprRuntimeVersion, daprDashboardVersion)
		verifyConfigs(t, daprPath)
	})

	t.Run("init with mariner images", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		output, err := cmdInit(daprRuntimeVersion, "--image-variant", "mariner")
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		daprPath := filepath.Join(homeDir, ".dapr")
		require.DirExists(t, daprPath, "Directory %s does not exist", daprPath)

		verifyContainers(t, daprRuntimeVersion+"-mariner")
		verifyBinaries(t, daprPath, daprRuntimeVersion, daprDashboardVersion)
		verifyConfigs(t, daprPath)
	})

	t.Run("init with --dapr-path flag", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		daprPath, err := os.MkdirTemp("", "dapr-e2e-init-with-flag-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath) // clean up

		output, err := cmdInit(daprRuntimeVersion, "--dapr-path", daprPath)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		verifyContainers(t, daprRuntimeVersion)
		verifyBinaries(t, daprPath, daprRuntimeVersion, daprDashboardVersion)
		verifyConfigs(t, daprPath)
	})

	t.Run("init with DAPR_PATH env var", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		daprPath, err := os.MkdirTemp("", "dapr-e2e-init-with-env-var-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath) // clean up

		os.Setenv("DAPR_PATH", daprPath)
		defer os.Unsetenv("DAPR_PATH")

		output, err := cmdInit(daprRuntimeVersion)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		verifyContainers(t, daprRuntimeVersion)
		verifyBinaries(t, daprPath, daprRuntimeVersion, daprDashboardVersion)
		verifyConfigs(t, daprPath)
	})

	t.Run("init with --dapr-path flag and DAPR_PATH env var", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		daprPath1, err := os.MkdirTemp("", "dapr-e2e-init-with-flag-and-env-1-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath1) // clean up
		daprPath2, err := os.MkdirTemp("", "dapr-e2e-init-with-flag-and-env-2-*")
		assert.NoError(t, err)
		defer os.RemoveAll(daprPath2) // clean up

		os.Setenv("DAPR_PATH", daprPath1)
		defer os.Unsetenv("DAPR_PATH")

		output, err := cmdInit(daprRuntimeVersion, "--dapr-path", daprPath2)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		verifyContainers(t, daprRuntimeVersion)
		verifyBinaries(t, daprPath2, daprRuntimeVersion, daprDashboardVersion)
		verifyConfigs(t, daprPath2)
	})
}

// verifyContainers ensures that the correct containers are up and running.
// Note, in case of slim installation, the containers are not installed and
// this test is automatically skipped.
func verifyContainers(t *testing.T, daprRuntimeVersion string) {
	t.Run("verifyContainers", func(t *testing.T) {
		if isSlimMode() {
			t.Log("Skipping container verification because of slim installation")
			return
		}

		cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)
		require.NoError(t, err)

		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
		require.NoError(t, err)

		daprContainers := map[string]string{
			"dapr_placement": daprRuntimeVersion,
			"dapr_zipkin":    "",
			"dapr_redis":     "",
		}

		for _, container := range containers {
			t.Logf("Found container: %v %s %s\n", container.Names, container.Image, container.State)
			if container.State != "running" {
				continue
			}
			name := strings.TrimPrefix(container.Names[0], "/")
			if expectedVersion, ok := daprContainers[name]; ok {
				if expectedVersion != "" {
					versionIndex := strings.LastIndex(container.Image, ":")
					if versionIndex == -1 {
						continue
					}
					version := container.Image[versionIndex+1:]
					if version != expectedVersion {
						continue
					}
				}

				delete(daprContainers, name)
			}
		}

		assert.Empty(t, daprContainers, "Missing containers: %v", daprContainers)
	})
}

// verifyBinaries ensures that the correct binaries are present in the correct path.
func verifyBinaries(t *testing.T, daprPath, runtimeVersion, dashboardVersion string) {
	binPath := filepath.Join(daprPath, "bin")
	require.DirExists(t, binPath, "Directory %s does not exist", binPath)

	binaries := map[string]string{
		"daprd":     runtimeVersion,
		"dashboard": dashboardVersion,
	}

	if isSlimMode() {
		binaries["placement"] = ""
	}

	for bin, version := range binaries {
		t.Run("verifyBinaries/"+bin, func(t *testing.T) {
			file := filepath.Join(binPath, bin)
			if runtime.GOOS == "windows" {
				file += ".exe"
			}
			require.FileExists(t, file, "File %s does not exist", file)

			if version != "" {
				output, err := spawn.Command(file, "--version")
				require.NoError(t, err, "failed to get version of %s", file)
				assert.Contains(t, output, version)
			}
		})
	}
}

// verifyConfigs ensures that the Dapr configuration and component YAMLs
// are present in the correct path and have the correct values.
func verifyConfigs(t *testing.T, daprPath string) {
	configSpec := map[interface{}]interface{}{}
	// tracing is not enabled in slim mode by default.
	if !isSlimMode() {
		configSpec = map[interface{}]interface{}{
			"tracing": map[interface{}]interface{}{
				"samplingRate": "1",
				"zipkin": map[interface{}]interface{}{
					"endpointAddress": "http://localhost:9411/api/v2/spans",
				},
			},
		}
	}

	configs := map[string]map[string]interface{}{
		"config.yaml": {
			"apiVersion": "dapr.io/v1alpha1",
			"kind":       "Configuration",
			"metadata": map[interface{}]interface{}{
				"name": "daprConfig",
			},
			"spec": configSpec,
		},
	}

	// The default components are not installed in slim mode.
	if !isSlimMode() {
		configs[filepath.Join("components", "statestore.yaml")] = map[string]interface{}{
			"apiVersion": "dapr.io/v1alpha1",
			"kind":       "Component",
			"metadata": map[interface{}]interface{}{
				"name": "statestore",
			},
			"spec": map[interface{}]interface{}{
				"type":    "state.redis",
				"version": "v1",
				"metadata": []interface{}{
					map[interface{}]interface{}{
						"name":  "redisHost",
						"value": "localhost:6379",
					},
					map[interface{}]interface{}{
						"name":  "redisPassword",
						"value": "",
					},
					map[interface{}]interface{}{
						"name":  "actorStateStore",
						"value": "true",
					},
				},
			},
		}
		configs[filepath.Join("components", "pubsub.yaml")] = map[string]interface{}{
			"apiVersion": "dapr.io/v1alpha1",
			"kind":       "Component",
			"metadata": map[interface{}]interface{}{
				"name": "pubsub",
			},
			"spec": map[interface{}]interface{}{
				"type":    "pubsub.redis",
				"version": "v1",
				"metadata": []interface{}{
					map[interface{}]interface{}{
						"name":  "redisHost",
						"value": "localhost:6379",
					},
					map[interface{}]interface{}{
						"name":  "redisPassword",
						"value": "",
					},
				},
			},
		}
	}

	for fileName, expected := range configs {
		t.Run("verifyConfigs/"+fileName, func(t *testing.T) {
			fullPath := filepath.Join(daprPath, fileName)
			bytes, err := os.ReadFile(fullPath)
			require.NoError(t, err, "failed to read file %s", fullPath)

			var actual map[string]interface{}
			err = yaml.Unmarshal(bytes, &actual)
			require.NoError(t, err, "failed to unmarshal file %s", fullPath)

			assert.Equal(t, expected, actual)
		})
	}
}
