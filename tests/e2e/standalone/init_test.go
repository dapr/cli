//go:build e2e && !template

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
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver"
	"github.com/dapr/cli/pkg/version"
	"github.com/dapr/cli/tests/e2e/common"
	"github.com/dapr/cli/tests/e2e/spawn"
	"github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestStandaloneInit(t *testing.T) {
	daprRuntimeVersion, daprDashboardVersion := common.GetVersionsFromEnv(t, false)

	t.Cleanup(func() {
		// remove dapr installation after all tests in this function.
		must(t, cmdUninstall, "failed to uninstall Dapr")
	})

	t.Run("init with invalid private registry", func(t *testing.T) {
		if isSlimMode() {
			t.Skip("Skipping init with private registry test because of slim installation")
		}

		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")
		args := []string{
			"--runtime-version", daprRuntimeVersion,
			"--dashboard-version", daprDashboardVersion,
			"--image-registry", "smplregistry.io/owner",
		}
		output, err := cmdInit(args...)
		t.Log(output)
		require.Error(t, err, "init failed")
	})

	t.Run("init should error if both --from-dir and --image-registry are given", func(t *testing.T) {
		if isSlimMode() {
			t.Skip("Skipping init with --image-registry and --from-dir test because of slim installation")
		}

		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")
		args := []string{
			"--runtime-version", daprRuntimeVersion,
			"--dashboard-version", daprDashboardVersion,
			"--image-registry", "localhost:5000",
			"--from-dir", "./local-dir",
		}
		output, err := cmdInit(args...)
		require.Error(t, err, "expected error if both flags are given")
		require.Contains(t, output, "both --image-registry and --from-dir flags cannot be given at the same time")
	})

	t.Run("init should error out if container runtime is not valid", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")
		args := []string{
			"--runtime-version", daprRuntimeVersion,
			"--dashboard-version", daprDashboardVersion,
			"--container-runtime", "invalid",
		}
		output, err := cmdInit(args...)
		require.Error(t, err, "expected error if container runtime is invalid")
		require.Contains(t, output, "Invalid container runtime")
	})

	t.Run("init", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		args := []string{
			"--runtime-version", daprRuntimeVersion,
			"--dashboard-version", daprDashboardVersion,
		}
		output, err := cmdInit(args...)
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

		args := []string{
			"--runtime-version", daprRuntimeVersion,
			"--dashboard-version", daprDashboardVersion,
			"--image-variant", "mariner",
		}
		output, err := cmdInit(args...)
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

	t.Run("init without runtime-version flag", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		output, err := cmdInit()
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		daprPath := filepath.Join(homeDir, ".dapr")
		require.DirExists(t, daprPath, "Directory %s does not exist", daprPath)

		latestDaprRuntimeVersion, err := version.GetDaprVersion()
		require.NoError(t, err)
		latestDaprDashboardVersion, err := version.GetDashboardVersion()
		require.NoError(t, err)

		verifyContainers(t, latestDaprRuntimeVersion)
		verifyBinaries(t, daprPath, latestDaprRuntimeVersion, latestDaprDashboardVersion)
		verifyConfigs(t, daprPath)

		placementPort := 50005
		if runtime.GOOS == "windows" {
			placementPort = 6050
		}

		verifyTCPLocalhost(t, placementPort)
	})

	t.Run("init version with scheduler", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		latestDaprRuntimeVersion, latestDaprDashboardVersion := common.GetVersionsFromEnv(t, true)

		args := []string{
			"--runtime-version", latestDaprRuntimeVersion,
			"--dev",
		}
		output, err := cmdInit(args...)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		daprPath := filepath.Join(homeDir, ".dapr")
		require.DirExists(t, daprPath, "Directory %s does not exist", daprPath)

		verifyContainers(t, latestDaprRuntimeVersion)
		verifyBinaries(t, daprPath, latestDaprRuntimeVersion, latestDaprDashboardVersion)
		verifyConfigs(t, daprPath)

		placementPort := 50005
		schedulerPort := 50006
		if runtime.GOOS == "windows" {
			placementPort = 6050
			schedulerPort = 6060
		}

		verifyTCPLocalhost(t, placementPort)
		verifyTCPLocalhost(t, schedulerPort)
	})

	t.Run("init with custom scheduler host", func(t *testing.T) {
		if isSlimMode() {
			t.Skip("Skipping scheduler host test because of slim installation")
		}

		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")

		customBroadcastHostPort := "192.168.42.42:50006"
		args := []string{
			"--scheduler-override-broadcast-host-port", customBroadcastHostPort,
		}
		output, err := cmdInit(args...)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		verifySchedulerBroadcastHostPort(t, customBroadcastHostPort)
	})

	t.Run("init without runtime-version flag with mariner images", func(t *testing.T) {
		// Ensure a clean environment
		must(t, cmdUninstall, "failed to uninstall Dapr")
		args := []string{
			"--image-variant", "mariner",
		}
		output, err := cmdInit(args...)
		t.Log(output)
		require.NoError(t, err, "init failed")
		assert.Contains(t, output, "Success! Dapr is up and running.")

		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		daprPath := filepath.Join(homeDir, ".dapr")
		require.DirExists(t, daprPath, "Directory %s does not exist", daprPath)

		latestDaprRuntimeVersion, err := version.GetDaprVersion()
		require.NoError(t, err)
		latestDaprDashboardVersion, err := version.GetDashboardVersion()
		require.NoError(t, err)

		verifyContainers(t, latestDaprRuntimeVersion+"-mariner")
		verifyBinaries(t, daprPath, latestDaprRuntimeVersion, latestDaprDashboardVersion)
		verifyConfigs(t, daprPath)
	})
}

// verifyContainers ensures that the correct containers are up and running.
// Note, in case of slim installation, the containers are not installed and
// this test is automatically skipped.
func verifyContainers(t *testing.T, daprRuntimeVersion string) {
	t.Helper()

	t.Run("verifyContainers", func(t *testing.T) {
		if isSlimMode() {
			t.Skip("Skipping container verification because of slim installation")
		}

		cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)
		require.NoError(t, err)

		containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
		require.NoError(t, err)

		daprContainers := map[string]string{
			"dapr_placement": daprRuntimeVersion,
			"dapr_zipkin":    "",
			"dapr_redis":     "",
		}

		v, err := semver.NewVersion(daprRuntimeVersion)
		require.NoError(t, err)
		if v.Major() >= 1 && v.Minor() >= 14 {
			daprContainers["dapr_scheduler"] = daprRuntimeVersion
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
	t.Helper()

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
			t.Helper()

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
	t.Helper()

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

// verifyTCPLocalhost verifies a given localhost TCP port is being listened to.
func verifyTCPLocalhost(t *testing.T, port int) {
	t.Helper()

	if isSlimMode() {
		t.Skip("Skipping container verification because of slim installation")
	}

	// Check that the server is up and can accept connections.
	endpoint := "127.0.0.1:" + strconv.Itoa(port)
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		conn, err := net.Dial("tcp", endpoint)
		//nolint:testifylint
		if assert.NoError(c, err) {
			conn.Close()
		}
	}, time.Second*10, time.Millisecond*10)
}

// verifySchedulerBroadcastHostPort verifies that the scheduler container was started with the correct broadcast host and port.
func verifySchedulerBroadcastHostPort(t *testing.T, expectedBroadcastHostPort string) {
	t.Helper()

	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv)
	require.NoError(t, err)

	containerInfo, err := cli.ContainerInspect(context.Background(), "dapr_scheduler")
	require.NoError(t, err)

	expectedArg := "--override-broadcast-host-port=" + expectedBroadcastHostPort
	assert.Contains(t, containerInfo.Args, expectedArg, "Expected scheduler argument %s not found in container args: %v", expectedArg, containerInfo.Args)
}
