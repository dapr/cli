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
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
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
	validFile := createTempFile(t, dirName, "valid_test_file.yaml")
	testcases := []struct {
		name        string
		input       string
		expectedErr bool
	}{
		{
			name:        "empty file path",
			input:       "",
			expectedErr: false,
		},
		{
			name:        "valid file path",
			input:       validFile,
			expectedErr: false,
		},
		{
			name:        "invalid file path",
			input:       "invalid_file_path",
			expectedErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual := ValidateFilePaths(tc.input)
			assert.Equal(t, tc.expectedErr, actual != nil)
		})
	}
}

func TestGetAbsPath(t *testing.T) {
	ex, err := os.Executable()
	assert.NoError(t, err)
	baseDir := filepath.Dir(ex)

	testcases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "relative path-1",
			input:    "./relative/path",
			expected: filepath.Join(baseDir, "relative", "path"),
		},
		{
			name:     "relative path-2",
			input:    "../relative/path",
			expected: filepath.Join(baseDir, "..", "relative", "path"),
		},
		{
			name:     "absolute path",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual := GetAbsPath(baseDir, tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestReadFile(t *testing.T) {
	fileName := createTempFile(t, "", "test_read_file")
	defer cleanupTempDir(t, fileName)
	testcases := []struct {
		name        string
		input       string
		expectedErr bool
	}{
		{
			name:        "empty file path",
			input:       "",
			expectedErr: true,
		},
		{
			name:        "valid file path",
			input:       fileName,
			expectedErr: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			_, actual := ReadFile(tc.input)
			assert.Equal(t, tc.expectedErr, actual != nil)
		})
	}
}

func TestIsYamlFile(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	validYamlPath := "valid.yaml"
	invalidFilePath := "invalid.json"
	dirPath := "test"
	_, err := fs.Create(validYamlPath)
	assert.NoError(t, err)
	_, err = fs.Create(invalidFilePath)
	assert.NoError(t, err)

	err = fs.Mkdir(dirPath, 0o755)
	assert.NoError(t, err)

	testcases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid yaml file path",
			input:    validYamlPath,
			expected: true,
		},
		{
			name:     "valid yml file path",
			input:    invalidFilePath,
			expected: false,
		},
		{
			name:     "valid yml file path",
			input:    dirPath,
			expected: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual := IsYAMLFile(tc.input, fs)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestFindFileInDir(t *testing.T) {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	invalidDir := "invalid_dir"
	validDirPath := "valid_dir"
	validDirWithNoDaprYAML := "valid_dir_no_dapr_yaml"

	err := fs.Mkdir(validDirPath, 0o755)
	assert.NoError(t, err)

	err = fs.Mkdir(validDirWithNoDaprYAML, 0o755)
	assert.NoError(t, err)

	_, err = fs.Create(filepath.Join(validDirPath, "dapr.yaml"))
	assert.NoError(t, err)
	_, err = fs.Create(filepath.Join(validDirPath, "test1.yaml"))
	assert.NoError(t, err)

	testcases := []struct {
		name             string
		input            string
		expectedErr      bool
		expectedFilePath string
	}{
		{
			name:             "valid yaml file path",
			input:            validDirPath,
			expectedErr:      false,
			expectedFilePath: filepath.Join(validDirPath, "dapr.yaml"),
		},
		{
			name:             "valid yml file path",
			input:            validDirWithNoDaprYAML,
			expectedErr:      true,
			expectedFilePath: "",
		},
		{
			name:             "valid yml file path",
			input:            invalidDir,
			expectedErr:      true,
			expectedFilePath: "",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			filePath, err := FindFileInDir(tc.input, "dapr.yaml", fs)
			assert.Equal(t, tc.expectedErr, err != nil)
			assert.Equal(t, tc.expectedFilePath, filePath)
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

func cleanupTempDir(t *testing.T, fileName string) {
	err := os.RemoveAll(fileName)
	assert.NoError(t, err)
}
