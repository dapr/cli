// +build e2e

// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone_test

import (
	"bufio"
	"context"
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

	"github.com/dapr/cli/tests/e2e/spawn"
	"github.com/dapr/go-sdk/service/common"
	daprHttp "github.com/dapr/go-sdk/service/http"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	daprRuntimeVersion   = "1.1.1"
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
		{"test run", testRun},
		{"test stop", testStop},
		{"test publish", testPublish},
		{"test invoke", testInvoke},
		{"test list", testList},
		{"test uninstall", testUninstall},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.phase)
	}
}

func TestNegativeScenarios(t *testing.T) {
	// Ensure a clean environment
	uninstall()
	daprPath := getDaprPath()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "expected no error on querying for os home dir")

	t.Run("run without install", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "run", "test")
		require.NoError(t, err, "expected no error status on run without install")
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

	t.Run("stop unkonwn flag", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "stop", "-p", "test")
		require.Error(t, err, "expected error on stop with unknown flag")
		require.Contains(t, output, "Error: unknown shorthand flag: 'p' in -p\nUsage:", "expected usage to be printed")
		require.Contains(t, output, "-a, --app-id string   The application id to be stopped", "expected usage to be printed")
	})

	t.Run("run unknown flags", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "run", "--flag")
		require.Error(t, err, "expected error on run unkonwn flag")
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

func testRun(t *testing.T) {
	daprPath := getDaprPath()

	t.Run("Normal exit", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "run", "--", "bash", "-c", "echo test")
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully")
		assert.Contains(t, output, "Exited Dapr successfully")
	})

	t.Run("Error exit", func(t *testing.T) {
		output, err := spawn.Command(daprPath, "run", "--", "bash", "-c", "exit 1")
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "The App process exited with error code: exit status 1")
		assert.Contains(t, output, "Exited Dapr successfully")
	})

	t.Run("API shutdown", func(t *testing.T) {
		// Test that the CLI exits on a daprd shutdown.
		output, err := spawn.Command(daprPath, "run", "--dapr-http-port", "9999", "--", "bash", "-c", "curl -v http://localhost:9999/v1.0/shutdown; sleep 10; exit 1")
		t.Log(output)
		require.NoError(t, err, "run failed")
		assert.Contains(t, output, "Exited App successfully", "App should be shutdown before it has a chance to return non-zero")
		assert.Contains(t, output, "Exited Dapr successfully")
	})

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
		listtOutputCheck(t, output)

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
	var sub = &common.Subscription{
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
	executeAgainstRunningDapr(t, func() {
		t.Run("publish from file", func(t *testing.T) {
			output, err := spawn.Command(daprPath, "publish", "--publish-app-id", "pub_e2e", "--pubsub", "pubsub", "--topic", "sample", "--data-file", "../testdata/message.json")
			t.Log(output)
			assert.NoError(t, err, "unable to publish from --data-file")
			assert.Contains(t, output, "Event published successfully")

			event := <-events
			assert.Equal(t, map[string]interface{}{"dapr": "is_great"}, event.Data)
		})

		t.Run("publish from string", func(t *testing.T) {
			output, err := spawn.Command(daprPath, "publish", "--publish-app-id", "pub_e2e", "--pubsub", "pubsub", "--topic", "sample", "--data", "{\"cli\": \"is_working\"}")
			t.Log(output)
			assert.NoError(t, err, "unable to publish from --data")
			assert.Contains(t, output, "Event published successfully")

			event := <-events
			assert.Equal(t, map[string]interface{}{"cli": "is_working"}, event.Data)
		})

		t.Run("publish from non-existant file fails", func(t *testing.T) {
			output, err := spawn.Command(daprPath, "publish", "--publish-app-id", "pub_e2e", "--pubsub", "pubsub", "--topic", "sample", "--data-file", "a/file/that/does/not/exist")
			t.Log(output)
			assert.Contains(t, output, "Error reading payload from 'a/file/that/does/not/exist'. Error: ")
			assert.Error(t, err, "a non-existant --data-file should fail with error")
		})

		t.Run("publish only one of data and data-file", func(t *testing.T) {
			output, err := spawn.Command(daprPath, "publish", "--publish-app-id", "pub_e2e", "--pubsub", "pubsub", "--topic", "sample", "--data-file", "../testdata/message.json", "--data", "{\"cli\": \"is_working\"}")
			t.Log(output)
			assert.Error(t, err, "--data and --data-file should not be allowed together")
			assert.Contains(t, output, "Only one of --data and --data-file allowed in the same publish command")

		})

		output, err := spawn.Command(getDaprPath(), "stop", "--app-id", "pub_e2e")
		t.Log(output)
		require.NoError(t, err, "dapr stop failed")
		assert.Contains(t, output, "app stopped successfully: pub_e2e")
	}, "run", "--app-id", "pub_e2e", "--app-port", "9988")

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
	executeAgainstRunningDapr(t, func() {
		t.Run("data from file", func(t *testing.T) {
			output, err := spawn.Command(daprPath, "invoke", "--app-id", "invoke_e2e", "--method", "test", "--data-file", "../testdata/message.json")
			t.Log(output)
			assert.NoError(t, err, "unable to invoke with  --data-file")
			assert.Contains(t, output, "App invoked successfully")
			assert.Contains(t, output, "{\"dapr\": \"is_great\"}")
		})

		t.Run("data from string", func(t *testing.T) {
			output, err := spawn.Command(daprPath, "invoke", "--app-id", "invoke_e2e", "--method", "test", "--data", "{\"cli\": \"is_working\"}")
			t.Log(output)
			assert.NoError(t, err, "unable to invoke with --data")
			assert.Contains(t, output, "{\"cli\": \"is_working\"}")
			assert.Contains(t, output, "App invoked successfully")
		})

		t.Run("data from non-existant file fails", func(t *testing.T) {
			output, err := spawn.Command(daprPath, "invoke", "--app-id", "invoke_e2e", "--method", "test", "--data-file", "a/file/that/does/not/exist")
			t.Log(output)
			assert.Error(t, err, "a non-existant --data-file should fail with error")
			assert.Contains(t, output, "Error reading payload from 'a/file/that/does/not/exist'. Error: ")
		})

		t.Run("invoke only one of data and data-file", func(t *testing.T) {
			output, err := spawn.Command(daprPath, "invoke", "--app-id", "invoke_e2e", "--method", "test", "--data-file", "../testdata/message.json", "--data", "{\"cli\": \"is_working\"}")
			t.Log(output)
			assert.Error(t, err, "--data and --data-file should not be allowed together")
			assert.Contains(t, output, "Only one of --data and --data-file allowed in the same invoke command")
		})

		output, err := spawn.Command(getDaprPath(), "stop", "--app-id", "invoke_e2e")
		t.Log(output)
		require.NoError(t, err, "dapr stop failed")
		assert.Contains(t, output, "app stopped successfully: invoke_e2e")
	}, "run", "--app-id", "invoke_e2e", "--app-port", "9987")

}

func listtOutputCheck(t *testing.T, output string) {
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
