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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandaloneUninstall(t *testing.T) {
	t.Run("uninstall should error out if container runtime is not valid", func(t *testing.T) {
		output, err := cmdUninstall("--container-runtime", "invalid")
		require.Error(t, err, "expected error if container runtime is invalid")
		require.Contains(t, output, "Invalid container runtime")
	})

	t.Run("uninstall", func(t *testing.T) {
		ensureDaprInstallation(t)

		output, err := cmdUninstall()
		t.Log(output)
		require.NoError(t, err, "dapr uninstall failed")
		assert.Contains(t, output, "Dapr has been removed successfully")

		// verify that .dapr directory does not exist.
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err, "failed to get user home directory")

		daprPath := filepath.Join(homeDir, ".dapr")
		require.NoDirExists(t, daprPath, "Directory %s does not exist", daprPath)

		verifyNoContainers(t)
	})
}

// verifyNoContainers verifies that no Dapr containers are running.
func verifyNoContainers(t *testing.T) {
	if isSlimMode() {
		t.Log("Skipping verifyNoContainers test in slim mode")
		return
	}

	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithVersion("1.48"))
	require.NoError(t, err)

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	require.NoError(t, err)

	// this is being stored as a map to allow easier deletes.
	daprContainers := map[string]bool{
		"dapr_placement": true,
		"dapr_zipkin":    true,
		"dapr_redis":     true,
	}

	for _, container := range containers {
		t.Logf("Found container %v %s %s\n", container.Names, container.Image, container.State)
		name := strings.TrimPrefix(container.Names[0], "/")
		// No deletes are expected since Dapr containers should not be running.
		delete(daprContainers, name)
	}

	// If any Dapr containers are still running after uninstall, this assertion will fail.
	assert.Equal(t, 3, len(daprContainers), "Found Dapr containers still running")
}
