// +build e2e

// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/dapr/cli/tests/e2e/spawn"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	daprNamespace        = "dapr-cli-tests"
	daprRuntimeVersion   = "1.0.1"
	daprDashboardVersion = "0.6.0"
)

func TestStandaloneInstall(t *testing.T) {
	// Ensure a clean environment
	uninstall()

	tests := []struct {
		name  string
		phase func(*testing.T)
	}{
		{"test install", testInstall},
		{"test uninstall", testUninstall},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.phase)
	}
}

func getDaprPath() string {
	distDir := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	return filepath.Join("..", "..", "..", "dist", distDir, "release", "dapr")
}

func uninstall() (string, error) {
	daprPath := getDaprPath()

	return spawn.Command(daprPath, "uninstall", "--all", "--log-as-json")
}

func testUninstall(t *testing.T) {
	output, err := uninstall()
	t.Log(output)
	require.NoError(t, err, "uninstall failed")

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	daprHome := filepath.Join(homeDir, ".dapr")
	_, err = os.Stat(daprHome)
	if assert.Error(t, err) {
		assert.True(t, os.IsNotExist(err), err.Error())
	}

	// Verify Containers

	cli, err := client.NewEnvClient()
	require.NoError(t, err)

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	require.NoError(t, err)

	notFound := map[string]string{
		"dapr_placement": daprRuntimeVersion,
		"dapr_zipkin":    "",
		"dapr_redis":     "",
	}

	for _, container := range containers {
		t.Logf("%v %s %s\n", container.Names, container.Image, container.State)
		name := strings.TrimPrefix(container.Names[0], "/")
		delete(notFound, name)
	}

	assert.Equal(t, 3, len(notFound))
}

func testInstall(t *testing.T) {
	daprPath := getDaprPath()

	output, err := spawn.Command(daprPath, "init", "--runtime-version", daprRuntimeVersion, "--log-as-json")
	t.Log(output)
	require.NoError(t, err, "init failed")

	// Verify Containers

	cli, err := client.NewEnvClient()
	require.NoError(t, err)

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	require.NoError(t, err)

	notFound := map[string]string{
		"dapr_placement": daprRuntimeVersion,
		"dapr_zipkin":    "",
		"dapr_redis":     "",
	}

	for _, container := range containers {
		t.Logf("%v %s %s\n", container.Names, container.Image, container.State)
		if container.State != "running" {
			continue
		}
		name := strings.TrimPrefix(container.Names[0], "/")
		if expectedVersion, ok := notFound[name]; ok {
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

			delete(notFound, name)
		}
	}

	assert.Empty(t, notFound)

	// Verify Binaries

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	path := filepath.Join(homeDir, ".dapr")
	binPath := filepath.Join(path, "bin")

	binaries := map[string]string{
		"daprd":     daprRuntimeVersion,
		"dashboard": daprDashboardVersion,
	}

	for bin, version := range binaries {
		t.Run(bin, func(t *testing.T) {
			file := filepath.Join(binPath, bin)
			if runtime.GOOS == "windows" {
				file += ".exe"
			}
			_, err := os.Stat(file)
			if !assert.NoError(t, err) {
				return
			}
			// Check version
			output, err := spawn.Command(file, "--version")
			if !assert.NoError(t, err) {
				return
			}
			output = strings.TrimSpace(output)
			if !assert.Equal(t, version, output) {
				return
			}
			delete(binaries, bin)
		})
	}

	assert.Empty(t, binaries)

	// Verify configs

	configs := map[string]map[string]interface{}{
		"config.yaml": {
			"apiVersion": "dapr.io/v1alpha1",
			"kind":       "Configuration",
			"metadata": map[interface{}]interface{}{
				"name": "daprConfig",
			},
			"spec": map[interface{}]interface{}{
				"tracing": map[interface{}]interface{}{
					"samplingRate": "1",
					"zipkin": map[interface{}]interface{}{
						"endpointAddress": "http://localhost:9411/api/v2/spans",
					},
				},
			},
		},
		filepath.Join("components", "statestore.yaml"): {
			"apiVersion": "dapr.io/v1alpha1",
			"kind":       "Component",
			"metadata": map[interface{}]interface{}{
				"name": "statestore",
			},
			"spec": map[interface{}]interface{}{
				"type":    "state.redis",
				"version": "v1.0",
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
		},
		filepath.Join("components", "pubsub.yaml"): {
			"apiVersion": "dapr.io/v1alpha1",
			"kind":       "Component",
			"metadata": map[interface{}]interface{}{
				"name": "pubsub",
			},
			"spec": map[interface{}]interface{}{
				"type":    "pubsub.redis",
				"version": "v1.0",
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
		},
	}

	for filename, contents := range configs {
		t.Run(filename, func(t *testing.T) {
			fullpath := filepath.Join(path, filename)
			contentBytes, err := ioutil.ReadFile(fullpath)
			if !assert.NoError(t, err) {
				return
			}
			var actual map[string]interface{}
			err = yaml.Unmarshal(contentBytes, &actual)
			if !assert.NoError(t, err) {
				return
			}
			if !assert.Equal(t, contents, actual) {
				return
			}

			delete(configs, filename)
		})
	}

	assert.Empty(t, configs)
}
