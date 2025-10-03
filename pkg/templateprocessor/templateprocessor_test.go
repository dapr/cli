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

package templateprocessor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSubstituteEnvVars(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		envVars        map[string]string
		expected       string
		expectTemplate bool
	}{
		{
			name:           "single env var substitution",
			input:          "host: {{REDIS_HOST}}",
			envVars:        map[string]string{"REDIS_HOST": "localhost"},
			expected:       "host: localhost",
			expectTemplate: true,
		},
		{
			name:           "multiple env var substitution",
			input:          "host: {{REDIS_HOST}}\nport: {{REDIS_PORT}}",
			envVars:        map[string]string{"REDIS_HOST": "localhost", "REDIS_PORT": "6379"},
			expected:       "host: localhost\nport: 6379",
			expectTemplate: true,
		},
		{
			name:           "env var not set - leave as template",
			input:          "host: {{MISSING_VAR}}",
			envVars:        map[string]string{},
			expected:       "host: {{MISSING_VAR}}",
			expectTemplate: false,
		},
		{
			name:           "mixed set and unset env vars",
			input:          "host: {{REDIS_HOST}}\nport: {{MISSING_PORT}}",
			envVars:        map[string]string{"REDIS_HOST": "localhost"},
			expected:       "host: localhost\nport: {{MISSING_PORT}}",
			expectTemplate: true,
		},
		{
			name:           "no templates",
			input:          "host: localhost\nport: 6379",
			envVars:        map[string]string{},
			expected:       "host: localhost\nport: 6379",
			expectTemplate: false,
		},
		{
			name:           "env var with underscores and numbers",
			input:          "key: {{MY_VAR_123}}",
			envVars:        map[string]string{"MY_VAR_123": "value123"},
			expected:       "key: value123",
			expectTemplate: true,
		},
		{
			name:           "lowercase should not match",
			input:          "key: {{lowercase_var}}",
			envVars:        map[string]string{"lowercase_var": "value"},
			expected:       "key: {{lowercase_var}}",
			expectTemplate: false,
		},
		{
			name:           "empty string substitution",
			input:          "key: {{EMPTY_VAR}}",
			envVars:        map[string]string{"EMPTY_VAR": ""},
			expected:       "key: ",
			expectTemplate: true,
		},
		{
			name:           "default value when env var not set",
			input:          "host: {{REDIS_HOST:localhost}}",
			envVars:        map[string]string{},
			expected:       "host: localhost",
			expectTemplate: true,
		},
		{
			name:           "env var overrides default value",
			input:          "host: {{REDIS_HOST:localhost}}",
			envVars:        map[string]string{"REDIS_HOST": "redis.example.com"},
			expected:       "host: redis.example.com",
			expectTemplate: true,
		},
		{
			name:           "default value with spaces",
			input:          "name: {{APP_NAME:My Application}}",
			envVars:        map[string]string{},
			expected:       "name: My Application",
			expectTemplate: true,
		},
		{
			name:           "default value with special characters",
			input:          "url: {{DATABASE_URL:postgresql://localhost:5432/db}}",
			envVars:        map[string]string{},
			expected:       "url: postgresql://localhost:5432/db",
			expectTemplate: true,
		},
		{
			name:           "empty default value",
			input:          "key: {{OPTIONAL_VAR:}}",
			envVars:        map[string]string{},
			expected:       "key: ",
			expectTemplate: true,
		},
		{
			name:           "multiple templates with defaults",
			input:          "host: {{HOST:localhost}}\nport: {{PORT:6379}}",
			envVars:        map[string]string{"HOST": "redis-server"},
			expected:       "host: redis-server\nport: 6379",
			expectTemplate: true,
		},
		{
			name:           "default value with colon in it",
			input:          "url: {{URL:http://localhost:8080}}",
			envVars:        map[string]string{},
			expected:       "url: http://localhost:8080",
			expectTemplate: true,
		},
		{
			name:           "template without default stays unchanged when var missing",
			input:          "key: {{MISSING_VAR}}",
			envVars:        map[string]string{},
			expected:       "key: {{MISSING_VAR}}",
			expectTemplate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables.
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result, hasTemplates := substituteEnvVars([]byte(tt.input))

			if string(result) != tt.expected {
				t.Errorf("substituteEnvVars() = %q, want %q", string(result), tt.expected)
			}

			if hasTemplates != tt.expectTemplate {
				t.Errorf("substituteEnvVars() hasTemplates = %v, want %v", hasTemplates, tt.expectTemplate)
			}
		})
	}
}

func TestShouldProcessFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"yaml file", "component.yaml", true},
		{"yml file", "component.yml", true},
		{"json file", "config.json", true},
		{"YAML uppercase", "component.YAML", true},
		{"text file", "readme.txt", false},
		{"go file", "main.go", false},
		{"no extension", "Dockerfile", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldProcessFile(tt.path)
			if result != tt.expected {
				t.Errorf("shouldProcessFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestProcessResourcesWithEnvVars(t *testing.T) {
	// Create a temporary directory with test files.
	tempDir := t.TempDir()

	// Create test directory structure.
	resourceDir := filepath.Join(tempDir, "resources")
	err := os.MkdirAll(resourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test component file with template.
	componentContent := `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
spec:
  type: state.redis
  metadata:
  - name: redisHost
    value: {{TEST_REDIS_HOST}}
  - name: redisPort
    value: {{TEST_REDIS_PORT}}
`
	componentPath := filepath.Join(resourceDir, "statestore.yaml")
	err = os.WriteFile(componentPath, []byte(componentContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test component file: %v", err)
	}

	// Create test config file without template.
	configContent := `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: appconfig
spec:
  tracing:
    samplingRate: "1"
`
	configPath := filepath.Join(resourceDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Create a non-processable file.
	readmePath := filepath.Join(resourceDir, "README.txt")
	err = os.WriteFile(readmePath, []byte("This is a readme"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test readme file: %v", err)
	}

	// Set environment variables.
	os.Setenv("TEST_REDIS_HOST", "localhost")
	os.Setenv("TEST_REDIS_PORT", "6379")
	defer os.Unsetenv("TEST_REDIS_HOST")
	defer os.Unsetenv("TEST_REDIS_PORT")

	// Process resources.
	result, err := ProcessResourcesWithEnvVars([]string{resourceDir})
	if err != nil {
		t.Fatalf("ProcessResourcesWithEnvVars() failed: %v", err)
	}
	defer Cleanup(result.TempDir)

	// Verify temp directory was created.
	if result.TempDir == "" {
		t.Error("TempDir is empty")
	}

	// Verify processed paths.
	if len(result.ProcessedPaths) != 1 {
		t.Errorf("Expected 1 processed path, got %d", len(result.ProcessedPaths))
	}

	// Verify templates were found.
	if !result.HasTemplates {
		t.Error("Expected HasTemplates to be true")
	}

	// Read processed component file.
	processedComponentPath := filepath.Join(result.ProcessedPaths[0], "statestore.yaml")
	processedContent, err := os.ReadFile(processedComponentPath)
	if err != nil {
		t.Fatalf("Failed to read processed component file: %v", err)
	}

	expectedContent := `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
spec:
  type: state.redis
  metadata:
  - name: redisHost
    value: localhost
  - name: redisPort
    value: 6379
`
	if string(processedContent) != expectedContent {
		t.Errorf("Processed content doesn't match.\nGot:\n%s\nWant:\n%s", string(processedContent), expectedContent)
	}

	// Verify config file was copied (no substitution).
	processedConfigPath := filepath.Join(result.ProcessedPaths[0], "config.yaml")
	processedConfigContent, err := os.ReadFile(processedConfigPath)
	if err != nil {
		t.Fatalf("Failed to read processed config file: %v", err)
	}

	if string(processedConfigContent) != configContent {
		t.Error("Config file content was modified when it shouldn't be")
	}

	// Verify non-processable file was copied.
	processedReadmePath := filepath.Join(result.ProcessedPaths[0], "README.txt")
	if _, err := os.Stat(processedReadmePath); os.IsNotExist(err) {
		t.Error("README.txt was not copied")
	}
}

func TestProcessResourcesWithMultiplePaths(t *testing.T) {
	// Create temporary directories with test files.
	tempDir := t.TempDir()

	// Create first resource directory.
	resourceDir1 := filepath.Join(tempDir, "resources1")
	err := os.MkdirAll(resourceDir1, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory 1: %v", err)
	}

	componentContent1 := `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: component1
spec:
  type: state.redis
  metadata:
  - name: host
    value: {{MULTI_TEST_HOST}}
`
	err = os.WriteFile(filepath.Join(resourceDir1, "component1.yaml"), []byte(componentContent1), 0644)
	if err != nil {
		t.Fatalf("Failed to create component1 file: %v", err)
	}

	// Create second resource directory.
	resourceDir2 := filepath.Join(tempDir, "resources2")
	err = os.MkdirAll(resourceDir2, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory 2: %v", err)
	}

	componentContent2 := `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: component2
spec:
  type: pubsub.redis
  metadata:
  - name: host
    value: {{MULTI_TEST_HOST}}
`
	err = os.WriteFile(filepath.Join(resourceDir2, "component2.yaml"), []byte(componentContent2), 0644)
	if err != nil {
		t.Fatalf("Failed to create component2 file: %v", err)
	}

	// Set environment variable.
	os.Setenv("MULTI_TEST_HOST", "redis-server")
	defer os.Unsetenv("MULTI_TEST_HOST")

	// Process multiple resource paths.
	result, err := ProcessResourcesWithEnvVars([]string{resourceDir1, resourceDir2})
	if err != nil {
		t.Fatalf("ProcessResourcesWithEnvVars() failed: %v", err)
	}
	defer Cleanup(result.TempDir)

	// Verify processed paths.
	if len(result.ProcessedPaths) != 2 {
		t.Errorf("Expected 2 processed paths, got %d", len(result.ProcessedPaths))
	}

	// Verify both files were processed.
	for i, processedPath := range result.ProcessedPaths {
		componentFile := filepath.Join(processedPath, fmt.Sprintf("component%d.yaml", i+1))
		content, err := os.ReadFile(componentFile)
		if err != nil {
			t.Errorf("Failed to read processed component%d file: %v", i+1, err)
			continue
		}

		if !contains(string(content), "redis-server") {
			t.Errorf("Component%d file does not contain substituted value", i+1)
		}
	}
}

func TestCleanup(t *testing.T) {
	// Create a temporary directory.
	tempDir, err := os.MkdirTemp("", "test-cleanup-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a file in it.
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Cleanup.
	err = Cleanup(tempDir)
	if err != nil {
		t.Errorf("Cleanup() failed: %v", err)
	}

	// Verify directory was removed.
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Error("Temp directory still exists after cleanup")
	}
}

func TestCleanupEmptyString(t *testing.T) {
	// Cleanup with empty string should not error.
	err := Cleanup("")
	if err != nil {
		t.Errorf("Cleanup(\"\") should not error, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}
