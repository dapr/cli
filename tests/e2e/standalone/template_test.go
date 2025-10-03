/*
Copyright 2025 The Dapr Authors
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

	"github.com/dapr/cli/pkg/templateprocessor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateProcessorIntegration(t *testing.T) {
	// Create a temporary directory for test resources.
	tempDir := t.TempDir()
	resourcesDir := filepath.Join(tempDir, "resources")
	err := os.MkdirAll(resourcesDir, 0755)
	require.NoError(t, err)

	// Create a component file with templates.
	componentContent := `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
spec:
  type: state.redis
  version: v1
  metadata:
  - name: redisHost
    value: {{TEST_REDIS_HOST}}
  - name: redisPort
    value: {{TEST_REDIS_PORT}}
  - name: enableTLS
    value: {{TEST_ENABLE_TLS}}
`
	componentPath := filepath.Join(resourcesDir, "statestore.yaml")
	err = os.WriteFile(componentPath, []byte(componentContent), 0644)
	require.NoError(t, err)

	// Set environment variables.
	os.Setenv("TEST_REDIS_HOST", "my-redis.example.com")
	os.Setenv("TEST_REDIS_PORT", "6380")
	os.Setenv("TEST_ENABLE_TLS", "true")
	defer os.Unsetenv("TEST_REDIS_HOST")
	defer os.Unsetenv("TEST_REDIS_PORT")
	defer os.Unsetenv("TEST_ENABLE_TLS")

	// Process resources.
	result, err := templateprocessor.ProcessResourcesWithEnvVars([]string{resourcesDir})
	require.NoError(t, err)
	require.NotNil(t, result)
	defer templateprocessor.Cleanup(result.TempDir)

	// Verify temp directory was created.
	assert.NotEmpty(t, result.TempDir)
	assert.True(t, result.HasTemplates)
	assert.Len(t, result.ProcessedPaths, 1)

	// Read and verify the processed component file.
	processedComponentPath := filepath.Join(result.ProcessedPaths[0], "statestore.yaml")
	processedContent, err := os.ReadFile(processedComponentPath)
	require.NoError(t, err)

	// Verify substitutions were made.
	processedStr := string(processedContent)
	assert.Contains(t, processedStr, "my-redis.example.com")
	assert.Contains(t, processedStr, "6380")
	assert.Contains(t, processedStr, "true")
	assert.NotContains(t, processedStr, "{{TEST_REDIS_HOST}}")
	assert.NotContains(t, processedStr, "{{TEST_REDIS_PORT}}")
	assert.NotContains(t, processedStr, "{{TEST_ENABLE_TLS}}")

	// Verify cleanup works.
	err = templateprocessor.Cleanup(result.TempDir)
	assert.NoError(t, err)

	// Verify temp directory was removed.
	_, err = os.Stat(result.TempDir)
	assert.True(t, os.IsNotExist(err))
}

func TestTemplateProcessorWithDefaultValues(t *testing.T) {
	// Create a temporary directory for test resources.
	tempDir := t.TempDir()
	resourcesDir := filepath.Join(tempDir, "resources")
	err := os.MkdirAll(resourcesDir, 0755)
	require.NoError(t, err)

	// Create a component file with default values.
	componentContent := `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
spec:
  type: state.redis
  version: v1
  metadata:
  - name: redisHost
    value: {{REDIS_HOST:localhost}}
  - name: redisPort
    value: {{REDIS_PORT:6379}}
  - name: enableTLS
    value: {{ENABLE_TLS:false}}
`
	componentPath := filepath.Join(resourcesDir, "statestore.yaml")
	err = os.WriteFile(componentPath, []byte(componentContent), 0644)
	require.NoError(t, err)

	// Set only one environment variable, others should use defaults.
	os.Setenv("REDIS_HOST", "my-redis.example.com")
	defer os.Unsetenv("REDIS_HOST")

	// Process resources.
	result, err := templateprocessor.ProcessResourcesWithEnvVars([]string{resourcesDir})
	require.NoError(t, err)
	require.NotNil(t, result)
	defer templateprocessor.Cleanup(result.TempDir)

	// Verify templates were processed.
	assert.True(t, result.HasTemplates)

	// Read the processed component file.
	processedComponentPath := filepath.Join(result.ProcessedPaths[0], "statestore.yaml")
	processedContent, err := os.ReadFile(processedComponentPath)
	require.NoError(t, err)

	// Verify: REDIS_HOST from env, others from defaults.
	processedStr := string(processedContent)
	assert.Contains(t, processedStr, "my-redis.example.com") // From env var
	assert.Contains(t, processedStr, "6379")                // From default
	assert.Contains(t, processedStr, "false")               // From default
	assert.NotContains(t, processedStr, "{{REDIS_HOST:")
	assert.NotContains(t, processedStr, "{{REDIS_PORT:")
	assert.NotContains(t, processedStr, "{{ENABLE_TLS:")
}

func TestTemplateProcessorWithMissingEnvVars(t *testing.T) {
	// Create a temporary directory for test resources.
	tempDir := t.TempDir()
	resourcesDir := filepath.Join(tempDir, "resources")
	err := os.MkdirAll(resourcesDir, 0755)
	require.NoError(t, err)

	// Create a component file with templates.
	componentContent := `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
spec:
  type: state.redis
  version: v1
  metadata:
  - name: redisHost
    value: {{MISSING_REDIS_HOST}}
  - name: redisPort
    value: 6379
`
	componentPath := filepath.Join(resourcesDir, "statestore.yaml")
	err = os.WriteFile(componentPath, []byte(componentContent), 0644)
	require.NoError(t, err)

	// Process resources without setting the env var.
	result, err := templateprocessor.ProcessResourcesWithEnvVars([]string{resourcesDir})
	require.NoError(t, err)
	require.NotNil(t, result)
	defer templateprocessor.Cleanup(result.TempDir)

	// Read the processed component file.
	processedComponentPath := filepath.Join(result.ProcessedPaths[0], "statestore.yaml")
	processedContent, err := os.ReadFile(processedComponentPath)
	require.NoError(t, err)

	// Verify template was left as-is since env var doesn't exist.
	processedStr := string(processedContent)
	assert.Contains(t, processedStr, "{{MISSING_REDIS_HOST}}")
	assert.Contains(t, processedStr, "6379")
}
