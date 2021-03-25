// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStandaloneConfig(t *testing.T) {
	testFile := "./test.yaml"

	t.Run("Standalone config", func(t *testing.T) {
		expectConfigZipkin := `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec:
  tracing:
    samplingRate: "1"
    zipkin:
      endpointAddress: http://test_zipkin_host:9411/api/v2/spans
`
		os.Remove(testFile)
		createDefaultConfiguration("test_zipkin_host", testFile)
		assert.FileExists(t, testFile)
		content, err := ioutil.ReadFile(testFile)
		assert.NoError(t, err)
		assert.Equal(t, expectConfigZipkin, string(content))
	})

	t.Run("Standalone config slim", func(t *testing.T) {
		expectConfigSlim := `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec: {}
`
		os.Remove(testFile)
		createDefaultConfiguration("", testFile)
		assert.FileExists(t, testFile)
		content, err := ioutil.ReadFile(testFile)
		assert.NoError(t, err)
		assert.Equal(t, expectConfigSlim, string(content))
	})

	os.Remove(testFile)
}

func TestRedisPubSubConfig(t *testing.T) {
	expectedPubSubConfig := `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: pubsub
spec:
  type: pubsub.redis
  version: v1.0
  metadata:
  - name: redisHost
    value: localhost:6379
  - name: redisPassword
    value: ""`
	tempDir := t.TempDir()
	createRedisPubSub("localhost", tempDir)
	pubsubFilePath := filepath.Join(tempDir, pubSubYamlFileName)
	assert.FileExists(t, pubsubFilePath)
	content, err := ioutil.ReadFile(pubsubFilePath)
	assert.NoError(t, err)
	assert.YAMLEq(t, expectedPubSubConfig, string(content))
}

func TestRedisStateStoreConfig(t *testing.T) {
	expectedStateStoreConfig := `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
spec:
  type: state.redis
  version: v1.0
  metadata:
  - name: redisHost
    value: localhost:6379
  - name: redisPassword
    value: ""
  - name: actorStateStore
    value: "true"`
	tempDir := t.TempDir()
	createRedisStateStore("localhost", tempDir)
	stateStoreFilePath := filepath.Join(tempDir, stateStoreYamlFileName)
	assert.FileExists(t, stateStoreFilePath)
	content, err := ioutil.ReadFile(stateStoreFilePath)
	assert.NoError(t, err)
	assert.YAMLEq(t, expectedStateStoreConfig, string(content))
}
