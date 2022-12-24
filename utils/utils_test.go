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

package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainerRuntimeUtils(t *testing.T) {
	testcases := []struct {
		name     string
		input    string
		expected string
		valid    bool
	}{
		{
			name:     "podman runtime is valid, and is returned as is",
			input:    "podman",
			expected: "podman",
			valid:    true,
		},
		{
			name:     "podman runtime with extra spaces is valid, and is returned as is",
			input:    "  podman  ",
			expected: "podman",
			valid:    true,
		},
		{
			name:     "docker runtime is valid, and is returned as is",
			input:    "docker",
			expected: "docker",
			valid:    true,
		},
		{
			name:     "docker runtime with extra spaces is valid, and is returned as is",
			input:    "     docker  ",
			expected: "docker",
			valid:    true,
		},
		{
			name:     "empty runtime is invalid, and docker is returned as default",
			input:    "",
			expected: "docker",
			valid:    false,
		},
		{
			name:     "invalid runtime is invalid, and docker is returned as default",
			input:    "invalid",
			expected: "docker",
			valid:    false,
		},
		{
			name:     "invalid runtime with extra spaces is invalid, and docker is returned as default",
			input:    "   invalid  ",
			expected: "docker",
			valid:    false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualValid := IsValidContainerRuntime(tc.input)
			if actualValid != tc.valid {
				t.Errorf("expected %v, got %v", tc.valid, actualValid)
			}

			actualCmd := GetContainerRuntimeCmd(tc.input)
			if actualCmd != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, actualCmd)
			}
		})
	}
}

func TestContains(t *testing.T) {
	testcases := []struct {
		name     string
		input    []string
		expected string
		valid    bool
	}{
		{
			name:     "empty list",
			input:    []string{},
			expected: "foo",
			valid:    false,
		},
		{
			name:     "list contains element",
			input:    []string{"foo", "bar", "baz"},
			expected: "foo",
			valid:    true,
		},
		{
			name:     "list does not contain element",
			input:    []string{"foo", "bar", "baz"},
			expected: "qux",
			valid:    false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualValid := Contains(tc.input, tc.expected)
			if actualValid != tc.valid {
				t.Errorf("expected %v, got %v", tc.valid, actualValid)
			}
		})
	}
}

func TestGetVersionAndImageVariant(t *testing.T) {
	testcases := []struct {
		name                 string
		input                string
		expectedVersion      string
		expectedImageVariant string
	}{
		{
			name:                 "image tag contains version and variant",
			input:                "1.9.0-mariner",
			expectedVersion:      "1.9.0",
			expectedImageVariant: "mariner",
		},
		{
			name:                 "image tag contains only version",
			input:                "1.9.0",
			expectedVersion:      "1.9.0",
			expectedImageVariant: "",
		},
		{
			name:                 "image tag contains only rc version and variant",
			input:                "1.9.0-rc.1-mariner",
			expectedVersion:      "1.9.0-rc.1",
			expectedImageVariant: "mariner",
		},
		{
			name:                 "image tag contains only rc version",
			input:                "1.9.0-rc.1",
			expectedVersion:      "1.9.0-rc.1",
			expectedImageVariant: "",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			version, imageVariant := GetVersionAndImageVariant(tc.input)
			assert.Equal(t, tc.expectedVersion, version)
			assert.Equal(t, tc.expectedImageVariant, imageVariant)
		})
	}
}

func TestValidateFilePaths(t *testing.T) {
	dirName := createTempDir(t, "test_validate_paths")
	defer cleanupTempDir(t, dirName)
	valideFile := createTempFile(t, dirName, "valid_test_file.yaml")
	testcases := []struct {
		name        string
		input       []string
		expectedErr bool
	}{
		{
			name:        "empty list",
			input:       []string{},
			expectedErr: false,
		},
		{
			name:        "list with valid file path",
			input:       []string{valideFile},
			expectedErr: false,
		},
		{
			name:        "list with invalid file path",
			input:       []string{"invalid_file_path"},
			expectedErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual := ValidateFilePaths(tc.input...)
			assert.Equal(t, tc.expectedErr, actual != nil)
		})
	}
}

func createTempDir(t *testing.T, tempDirName string) string {
	dirName, err := os.MkdirTemp("", tempDirName)
	assert.NoError(t, err)
	return dirName
}

func createTempFile(t *testing.T, tempDirName, fileName string) string {
	file, err := os.CreateTemp(tempDirName, fileName)
	assert.NoError(t, err)
	defer file.Close()
	return file.Name()
}

func cleanupTempDir(t *testing.T, dirName string) {
	err := os.RemoveAll(dirName)
	assert.NoError(t, err)
}
