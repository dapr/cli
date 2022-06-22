//go:build e2e
// +build e2e

/*
Copyright 2021 The Dapr Authors
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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	testCommon "github.com/dapr/cli/tests/e2e/common"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/dapr/cli/tests/e2e/spawn"
	"github.com/dapr/go-sdk/service/common"
	daprHttp "github.com/dapr/go-sdk/service/http"
)

var (
	daprRuntimeVersion   string
	daprDashboardVersion string
)

var socketCases = []string{"", "/tmp"}

func TestStandaloneInstall(t *testing.T) {
	// Ensure a clean environment.
	uninstall()
	daprRuntimeVersion, daprDashboardVersion = testCommon.GetVersionsFromEnv(t)

	tests := []struct {
		name  string
		phase func(*testing.T)
	}{
		{"test install", testInstall},
		{"test run log json enabled", testRunLogJSON},
		{"test run", testRun},
		{"test stop", testStop},
		{"test publish", testPublish},
		{"test invoke", testInvoke},
		{"test list", testList},
		{"test uninstall", testUninstall},
		{"test version", testVersion},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.phase)
	}
}

func TestEnableAPILogging(t *testing.T) {
	// Ensure a clean environment.
	uninstall()
	daprRuntimeVersion, daprDashboardVersion = testCommon.GetVersionsFromEnv(t)

	tests := []struct {
		name  string
		phase func(*testing.T)
	}{
		{"test install", testInstall},
		{"test run enable api logging", testRunEnableAPILogging},
		{"test uninstall", testUninstall},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.phase)
	}
}

func TestNegativeScenarios(t *testing.T) {
	// Ensure a clean environment
	uninstall()
	daprRuntimeVersion, daprDashboardVersion = testCommon.GetVersionsFromEnv(t)
	daprPath := getDaprPath()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "expected no error on querying for os home dir")

	t.Run("run without install", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "run", "test")
		require.Error(t, err, "expected error status on run without install")
		path := filepath.Join(homeDir, ".dapr", "components")
		require.Contains(t, output, path+": no such file or directory", "expected output to contain message")
	})

	t.Run("list without install", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "list")
		require.NoError(t, err, "expected no error status on list without install")
		require.Equal(t, "No Dapr instances found.\n", output)
	})

	t.Run("stop without install", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "stop", "-a", "test")
		require.NoError(t, err, "expected no error on stop without install")
		require.Contains(t, output, "failed to stop app id test: couldn't find app id test", "expected output to match")
	})

	t.Run("stop unknown flag", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "stop", "-p", "test")
		require.Error(t, err, "expected error on stop with unknown flag")
		require.Contains(t, output, "Error: unknown shorthand flag: 'p' in -p\nUsage:", "expected usage to be printed")
		require.Contains(t, output, "-a, --app-id string   The application id to be stopped", "expected usage to be printed")
	})

	t.Run("run unknown flags", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "run", "--flag")
		require.Error(t, err, "expected error on run unknown flag")
		require.Contains(t, output, "Error: unknown flag: --flag\nUsage:", "expected usage to be printed")
		require.Contains(t, output, "-a, --app-id string", "expected usage to be printed")
		require.Contains(t, output, "The id for your application, used for service discovery", "expected usage to be printed")
	})

	t.Run("uninstall without install", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "uninstall", "--all")
		require.NoError(t, err, "expected no error on uninstall without install")
		require.Contains(t, output, "Removing Dapr from your machine...", "expected output to contain message")
		path := filepath.Join(homeDir, ".dapr", "bin")
		require.Contains(t, output, "WARNING: "+path+" does not exist", "expected output to contain message")
		require.Contains(t, output, "WARNING: dapr_placement container does not exist", "expected output to contain message")
		require.Contains(t, output, "WARNING: dapr_redis container does not exist", "expected output to contain message")
		require.Contains(t, output, "WARNING: dapr_zipkin container does not exist", "expected output to contain message")
		path = filepath.Join(homeDir, ".dapr")
		require.Contains(t, output, "WARNING: "+path+" does not exist", "expected output to contain message")
		require.Contains(t, output, "Dapr has been removed successfully")
	})

	t.Run("filter dashboard instance from list", func(t *testing.T) {
		spawn.Command(daprPath, "dashboard", "-p", "5555")
		output, err := spawn.Command(daprPath, "list")
		require.NoError(t, err, "expected no error status on list without install")
		require.Equal(t, "No Dapr instances found.\n", output)
	})

	t.Run("error if both --from-dir and --image-registry given", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "init", "--image-registry", "localhost:5000", "--from-dir", "./local-dir")
		require.Error(t, err, "expected error if both flags are given")
		require.Contains(t, output, "both --image-registry and --from-dir flags cannot be given at the same time")
	})
}

func TestPrivateRegistry(t *testing.T) {
	// Ensure a clean environment.
	uninstall()
	daprRuntimeVersion, daprDashboardVersion = testCommon.GetVersionsFromEnv(t)

	tests := []struct {
		name  string
		phase func(*testing.T)
	}{
		{"test install fails", testInstallWithPrivateRegsitry},
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

	// verify all artifacts(conatiners, binaries, configs) after successfull install
	verifyArtifactsAfterInstall(t)
}

func testInstallWithPrivateRegsitry(t *testing.T) {
	daprPath := getDaprPath()
	output, err := spawn.Command(daprPath, "init", "--runtime-version", daprRuntimeVersion, "--image-registry", "smplregistry.io/owner", "--log-as-json")
	t.Log(output)
	require.Error(t, err, "init failed")
}

func verifyArtifactsAfterInstall(t *testing.T) {
	// Verify Containers
	cli, err := client.NewClientWithOpts(client.FromEnv)
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
			// Changing version check since there is a log that is output on daprd --version
			// 2021/11/12 11:10:38 maxprocs: Leaving GOMAXPROCS=12: CPU quota undefined
			// before the version is output
			if !assert.Contains(t, output, version) {
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
		},
		filepath.Join("components", "pubsub.yaml"): {
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

func testRunLogJSON(t *testing.T) {
	daprPath := getDaprPath()

	t.Run(fmt.Sprintf("check JSON log"), func(t *testing.T) {
		output, err := spawn.Command(daprPath, "run", "--app-id", "logjson", "--log-as-json", "--", "bash", "-c", "echo 'test'")
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "{\"app_id\":\"logjson\"")
		assert.Contains(t, output, "\"type\":\"log\"")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
	})
}

func testRun(t *testing.T) {
	daprPath := getDaprPath()

	for _, path := range socketCases {
		t.Run(fmt.Sprintf("normal exit, socket: %s", path), func(t *testing.T) {
			output, err := spawn.Command(daprPath, "run", "--unix-domain-socket", path, "--", "bash", "-c", "echo test")
			t.Log(output)
			require.NoError(t, err, "run failed")
			assert.Contains(t, output, "Exited App successfully")
			assert.Contains(t, output, "Exited Dapr successfully")
		})

		t.Run(fmt.Sprintf("error exit, socket: %s", path), func(t *testing.T) {
			output, err := spawn.Command(daprPath, "run", "--unix-domain-socket", path, "--", "bash", "-c", "exit 1")
			t.Log(output)
			require.Error(t, err, "run failed")
			assert.Contains(t, output, "The App process exited with error code: exit status 1")
			assert.Contains(t, output, "Exited Dapr successfully")
		})

	}

	t.Run("API shutdown without socket", func(t *testing.T) {
		// Test that the CLI exits on a daprd shutdown.
		output, err := spawn.Command(daprPath, "run", "--dapr-http-port", "9999", "--", "bash", "-c", "curl -v -X POST http://localhost:9999/v1.0/shutdown; sleep 10; exit 1")
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully", "App should be shutdown before it has a chance to return non-zero")
		assert.Contains(t, output, "Exited Dapr successfully")
	})

	t.Run("API shutdown with socket", func(t *testing.T) {
		// Test that the CLI exits on a daprd shutdown.
		output, err := spawn.Command(daprPath, "run", "--app-id", "testapp", "--unix-domain-socket", "/tmp", "--", "bash", "-c", "curl --unix-socket /tmp/dapr-testapp-http.socket -v -X POST http://unix/v1.0/shutdown; sleep 10; exit 1")
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited Dapr successfully")
	})
}

func testRunEnableAPILogging(t *testing.T) {
	daprPath := getDaprPath()
	args := []string{
		"run",
		"--app-id", "enableApiLogging_info",
		"--enable-api-logging",
		"--log-level", "info",
		"--", "bash", "-c", "echo 'test'",
	}

	t.Run(fmt.Sprintf("check enableAPILogging flag in enabled mode"), func(t *testing.T) {
		output, err := spawn.Command(daprPath, args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "level=info msg=\"HTTP API Called: PUT /v1.0/metadata/appCommand\"")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
	})

	args = []string{
		"run",
		"--app-id", "enableApiLogging_info",
		"--", "bash", "-c", "echo 'test'",
	}

	t.Run(fmt.Sprintf("check enableAPILogging flag in disabled mode"), func(t *testing.T) {
		output, err := spawn.Command(daprPath, args...)
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
		assert.NotContains(t, output, "level=info msg=\"HTTP API Called: PUT /v1.0/metadata/appCommand\"")
	})
}	

func testVersion(t *testing.T) {
	daprPath := getDaprPath()

	output, err := spawn.Command(daprPath, "version")
	t.Log(output)
	require.NoError(t, err, "dapr version failed")
	versionOutputCheck(t, output)

	output, err = spawn.Command(getDaprPath(), "version", "-o", "json")
	t.Log(output)
	require.NoError(t, err, "dapr version failed")
	versionJsonOutputCheck(t, output)
}

func versionOutputCheck(t *testing.T, output string) {
	lines := strings.Split(output, "\n")
	assert.GreaterOrEqual(t, len(lines), 2, "expected at least 2 fields in components outptu")
	assert.Contains(t, lines[0], "CLI version")
	assert.Contains(t, lines[1], "Runtime version")
}

func versionJsonOutputCheck(t *testing.T, output string) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	assert.NoError(t, err, "output was not valid JSON")
}

func executeAgainstRunningDapr(t *testing.T, f func(), daprArgs ...string) {
	daprPath := getDaprPath()

	cmd := exec.Command(daprPath, daprArgs...)
	reader, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(reader)

	cmd.Start()

	daprOutput := ""
	for scanner.Scan() {
		outputChunk := scanner.Text()
		t.Log(outputChunk)
		if strings.Contains(outputChunk, "You're up and running!") {
			f()
		}
		daprOutput += outputChunk
	}

	err := cmd.Wait()
	require.NoError(t, err, "dapr didn't exit cleanly")
	assert.NotContains(t, daprOutput, "The App process exited with error code: exit status", "Stop command should have been called before the app had a chance to exit")
	assert.Contains(t, daprOutput, "Exited Dapr successfully")
}

func testList(t *testing.T) {
	executeAgainstRunningDapr(t, func() {
		output, err := spawn.Command(getDaprPath(), "list")
		t.Log(output)
		require.NoError(t, err, "dapr list failed")
		listOutputCheck(t, output)

		output, err = spawn.Command(getDaprPath(), "list", "-o", "table")
		t.Log(output)
		require.NoError(t, err, "dapr list failed")
		listOutputCheck(t, output)

		output, err = spawn.Command(getDaprPath(), "list", "-o", "json")
		t.Log(output)
		require.NoError(t, err, "dapr list failed")
		listJsonOutputCheck(t, output)

		output, err = spawn.Command(getDaprPath(), "list", "-o", "yaml")
		t.Log(output)
		require.NoError(t, err, "dapr list failed")
		listYamlOutputCheck(t, output)

		output, err = spawn.Command(getDaprPath(), "list", "-o", "invalid")
		t.Log(output)
		require.Error(t, err, "dapr list should fail with an invalid output format")

		// We can call stop so as not to wait for the app to time out
		output, err = spawn.Command(getDaprPath(), "stop", "--app-id", "dapr_e2e_list")
		t.Log(output)
		require.NoError(t, err, "dapr stop failed")
		assert.Contains(t, output, "app stopped successfully: dapr_e2e_list")
	}, "run", "--app-id", "dapr_e2e_list", "-H", "3555", "-G", "4555", "--", "bash", "-c", "sleep 10 ; exit 0")
}

func testStop(t *testing.T) {
	executeAgainstRunningDapr(t, func() {
		output, err := spawn.Command(getDaprPath(), "stop", "--app-id", "dapr_e2e_stop")
		t.Log(output)
		require.NoError(t, err, "dapr stop failed")
		assert.Contains(t, output, "app stopped successfully: dapr_e2e_stop")
	}, "run", "--app-id", "dapr_e2e_stop", "--", "bash", "-c", "sleep 60 ; exit 1")
}

func testPublish(t *testing.T) {
	sub := &common.Subscription{
		PubsubName: "pubsub",
		Topic:      "sample",
		Route:      "/orders",
	}

	s := daprHttp.NewService(":9988")

	events := make(chan *common.TopicEvent)

	err := s.AddTopicEventHandler(sub, func(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
		events <- e
		return false, nil
	})

	assert.NoError(t, err, "unable to AddTopicEventHandler")

	defer s.Stop()
	go func() {
		err = s.Start()

		assert.NoError(t, err, "unable to listen on :9988")
	}()

	daprPath := getDaprPath()
	for _, path := range socketCases {
		executeAgainstRunningDapr(t, func() {
			t.Run(fmt.Sprintf("publish message from file with socket %s", path), func(t *testing.T) {
				output, err := spawn.Command(daprPath, "publish", "--publish-app-id", "pub_e2e", "--unix-domain-socket", path, "--pubsub", "pubsub", "--topic", "sample", "--data-file", "../testdata/message.json")
				t.Log(output)
				assert.NoError(t, err, "unable to publish from --data-file")
				assert.Contains(t, output, "Event published successfully")

				event := <-events
				assert.Equal(t, map[string]interface{}{"dapr": "is_great"}, event.Data)
			})

			t.Run(fmt.Sprintf("publish cloudevent from file with socket %s", path), func(t *testing.T) {
				output, err := spawn.Command(daprPath, "publish", "--publish-app-id", "pub_e2e", "--unix-domain-socket", path, "--pubsub", "pubsub", "--topic", "sample", "--data-file", "../testdata/cloudevent.json")
				t.Log(output)
				assert.NoError(t, err, "unable to publish from --data-file")
				assert.Contains(t, output, "Event published successfully")

				event := <-events
				assert.Equal(t, &common.TopicEvent{
					ID:              "3cc97064-edd1-49f4-b911-c959a7370e68",
					Source:          "e2e_test",
					SpecVersion:     "1.0",
					Type:            "test.v1",
					DataContentType: "application/json",
					Subject:         "e2e_subject",
					PubsubName:      "pubsub",
					Topic:           "sample",
					Data:            map[string]interface{}{"dapr": "is_great"},
				}, event)
			})

			t.Run(fmt.Sprintf("publish from string with socket %s", path), func(t *testing.T) {
				output, err := spawn.Command(daprPath, "publish", "--publish-app-id", "pub_e2e", "--unix-domain-socket", path, "--pubsub", "pubsub", "--topic", "sample", "--data", "{\"cli\": \"is_working\"}")
				t.Log(output)
				assert.NoError(t, err, "unable to publish from --data")
				assert.Contains(t, output, "Event published successfully")

				event := <-events
				assert.Equal(t, map[string]interface{}{"cli": "is_working"}, event.Data)
			})

			t.Run(fmt.Sprintf("publish from non-existent file fails with socket %s", path), func(t *testing.T) {
				output, err := spawn.Command(daprPath, "publish", "--publish-app-id", "pub_e2e", "--unix-domain-socket", path, "--pubsub", "pubsub", "--topic", "sample", "--data-file", "a/file/that/does/not/exist")
				t.Log(output)
				assert.Contains(t, output, "Error reading payload from 'a/file/that/does/not/exist'. Error: ")
				assert.Error(t, err, "a non-existent --data-file should fail with error")
			})

			t.Run(fmt.Sprintf("publish only one of data and data-file with socket %s", path), func(t *testing.T) {
				output, err := spawn.Command(daprPath, "publish", "--publish-app-id", "pub_e2e", "--unix-domain-socket", path, "--pubsub", "pubsub", "--topic", "sample", "--data-file", "../testdata/message.json", "--data", "{\"cli\": \"is_working\"}")
				t.Log(output)
				assert.Error(t, err, "--data and --data-file should not be allowed together")
				assert.Contains(t, output, "Only one of --data and --data-file allowed in the same publish command")
			})

			output, err := spawn.Command(getDaprPath(), "stop", "--app-id", "pub_e2e")
			t.Log(output)
			require.NoError(t, err, "dapr stop failed")
			assert.Contains(t, output, "app stopped successfully: pub_e2e")
		}, "run", "--app-id", "pub_e2e", "--app-port", "9988", "--unix-domain-socket", path)
	}
}

func testInvoke(t *testing.T) {
	s := daprHttp.NewService(":9987")

	err := s.AddServiceInvocationHandler("/test", func(ctx context.Context, e *common.InvocationEvent) (*common.Content, error) {
		val := &common.Content{
			Data:        e.Data,
			ContentType: e.ContentType,
			DataTypeURL: e.DataTypeURL,
		}
		return val, nil
	})

	assert.NoError(t, err, "unable to AddTopicEventHandler")

	defer s.Stop()
	go func() {
		err = s.Start()

		assert.NoError(t, err, "unable to listen on :9987")
	}()

	daprPath := getDaprPath()
	for _, path := range socketCases {
		executeAgainstRunningDapr(t, func() {
			t.Run(fmt.Sprintf("data from file with socket %s", path), func(t *testing.T) {
				output, err := spawn.Command(daprPath, "invoke", "--app-id", "invoke_e2e", "--unix-domain-socket", path, "--method", "test", "--data-file", "../testdata/message.json")
				t.Log(output)
				assert.NoError(t, err, "unable to invoke with  --data-file")
				assert.Contains(t, output, "App invoked successfully")
				assert.Contains(t, output, "{\"dapr\": \"is_great\"}")
			})

			t.Run(fmt.Sprintf("data from string with socket %s", path), func(t *testing.T) {
				output, err := spawn.Command(daprPath, "invoke", "--app-id", "invoke_e2e", "--unix-domain-socket", path, "--method", "test", "--data", "{\"cli\": \"is_working\"}")
				t.Log(output)
				assert.NoError(t, err, "unable to invoke with --data")
				assert.Contains(t, output, "{\"cli\": \"is_working\"}")
				assert.Contains(t, output, "App invoked successfully")
			})

			t.Run(fmt.Sprintf("data from non-existent file fails with socket %s", path), func(t *testing.T) {
				output, err := spawn.Command(daprPath, "invoke", "--app-id", "invoke_e2e", "--unix-domain-socket", path, "--method", "test", "--data-file", "a/file/that/does/not/exist")
				t.Log(output)
				assert.Error(t, err, "a non-existent --data-file should fail with error")
				assert.Contains(t, output, "Error reading payload from 'a/file/that/does/not/exist'. Error: ")
			})

			t.Run(fmt.Sprintf("invoke only one of data and data-file with socket %s", path), func(t *testing.T) {
				output, err := spawn.Command(daprPath, "invoke", "--app-id", "invoke_e2e", "--unix-domain-socket", path, "--method", "test", "--data-file", "../testdata/message.json", "--data", "{\"cli\": \"is_working\"}")
				t.Log(output)
				assert.Error(t, err, "--data and --data-file should not be allowed together")
				assert.Contains(t, output, "Only one of --data and --data-file allowed in the same invoke command")
			})

			t.Run(fmt.Sprintf("invoke an invalid app %s", path), func(t *testing.T) {
				output, err := spawn.Command(daprPath, "invoke", "--app-id", "invoke_e2e_2", "--unix-domain-socket", path, "--method", "test")
				t.Log(output)
				assert.Error(t, err, "app invoke_e2e_2 should not exist")
				assert.Contains(t, output, "error invoking app invoke_e2e_2: app ID invoke_e2e_2 not found")
			})

			t.Run(fmt.Sprintf("invoke with an invalid method name %s", path), func(t *testing.T) {
				output, err := spawn.Command(daprPath, "invoke", "--app-id", "invoke_e2e", "--unix-domain-socket", path, "--method", "test2")
				t.Log(output)
				assert.Error(t, err, "method test2 should not exist")
				assert.Contains(t, output, "error invoking app invoke_e2e: 404 Not Found")
			})

			output, err := spawn.Command(getDaprPath(), "stop", "--app-id", "invoke_e2e")
			t.Log(output)
			require.NoError(t, err, "dapr stop failed")
			assert.Contains(t, output, "app stopped successfully: invoke_e2e")
		}, "run", "--app-id", "invoke_e2e", "--app-port", "9987", "--unix-domain-socket", path)
	}
}

func listOutputCheck(t *testing.T, output string) {
	lines := strings.Split(output, "\n")[1:] // remove header
	// only one app is runnning at this time
	fields := strings.Fields(lines[0])
	// Fields splits on space, so Created time field might be split again
	assert.GreaterOrEqual(t, len(fields), 4, "expected at least 4 fields in components outptu")
	assert.Equal(t, "dapr_e2e_list", fields[0], "expected name to match")
	assert.Equal(t, "3555", fields[1], "expected http port to match")
	assert.Equal(t, "4555", fields[2], "expected grpc port to match")
	assert.Equal(t, "0", fields[3], "expected app port to match")
}

func listJsonOutputCheck(t *testing.T, output string) {
	var result map[string]interface{}

	err := json.Unmarshal([]byte(output), &result)

	assert.NoError(t, err, "output was not valid JSON")

	assert.Equal(t, "dapr_e2e_list", result["appId"], "expected name to match")
	assert.Equal(t, 3555, int(result["httpPort"].(float64)), "expected http port to match")
	assert.Equal(t, 4555, int(result["grpcPort"].(float64)), "expected grpc port to match")
	assert.Equal(t, 0, int(result["appPort"].(float64)), "expected app port to match")
}

func listYamlOutputCheck(t *testing.T, output string) {
	var result map[string]interface{}

	err := yaml.Unmarshal([]byte(output), &result)

	assert.NoError(t, err, "output was not valid YAML")

	assert.Equal(t, "dapr_e2e_list", result["appId"], "expected name to match")
	assert.Equal(t, 3555, result["httpPort"], "expected http port to match")
	assert.Equal(t, 4555, result["grpcPort"], "expected grpc port to match")
	assert.Equal(t, 0, result["appPort"], "expected app port to match")
}
