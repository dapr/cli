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
	"bytes"
	"math"
	"os"
	"path/filepath"
	"runtime"
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := ValidateFilePath(tc.input)
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := GetAbsPath(baseDir, tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestResolveHomeDir(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)

	testcases := []struct {
		name        string
		input       string
		expected    string
		skipWindows bool
	}{
		{
			name:        "empty path",
			input:       "",
			expected:    "",
			skipWindows: false,
		},
		{
			name:        "home directory prefix with ~/",
			input:       "~/home/path",
			expected:    filepath.Join(homeDir, "home", "path"),
			skipWindows: true,
		},
		{
			name:        "home directory prefix with ~/.",
			input:       "~/./home/path",
			expected:    filepath.Join(homeDir, ".", "home", "path"),
			skipWindows: true,
		},
		{
			name:        "home directory prefix with ~/..",
			input:       "~/../home/path",
			expected:    filepath.Join(homeDir, "..", "home", "path"),
			skipWindows: true,
		},
		{
			name:        "no home directory prefix",
			input:       "../home/path",
			expected:    "../home/path",
			skipWindows: false,
		},
		{
			name:        "absolute path",
			input:       "/absolute/path",
			expected:    "/absolute/path",
			skipWindows: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipWindows && runtime.GOOS == "windows" {
				t.Skip("Skipping test on Windows")
			}
			t.Parallel()
			actual, err := ResolveHomeDir(tc.input)
			assert.NoError(t, err)
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, actual := ReadFile(tc.input)
			assert.Equal(t, tc.expectedErr, actual != nil)
		})
	}
}

// Following is the directory and file structure created for this test in the os's default temp directory:
// test_find_file_in_dir/valid_dir/dapr.yaml.
// test_find_file_in_dir/valid_dir/test1.yaml.
// test_find_file_in_dir/valid_dir_no_dapr_yaml.
func TestFindFileInDir(t *testing.T) {
	nonExistentDirName := "invalid_dir"
	validDirNameWithDaprYAMLFile := "valid_dir"
	validDirWithNoDaprYAML := "valid_dir_no_dapr_yaml"

	dirName := createTempDir(t, "test_find_file_in_dir")
	t.Cleanup(func() {
		cleanupTempDir(t, dirName)
	})

	err := os.Mkdir(filepath.Join(dirName, validDirNameWithDaprYAMLFile), 0o755)
	assert.NoError(t, err)

	err = os.Mkdir(filepath.Join(dirName, validDirWithNoDaprYAML), 0o755)
	assert.NoError(t, err)

	fl, err := os.Create(filepath.Join(dirName, validDirNameWithDaprYAMLFile, "dapr.yaml"))
	assert.NoError(t, err)
	fl.Close()

	fl, err = os.Create(filepath.Join(dirName, validDirNameWithDaprYAMLFile, "test1.yaml"))
	assert.NoError(t, err)
	fl.Close()

	testcases := []struct {
		name             string
		input            string
		expectedErr      bool
		expectedFilePath string
	}{
		{
			name:             "valid directory path with dapr.yaml file",
			input:            filepath.Join(dirName, validDirNameWithDaprYAMLFile),
			expectedErr:      false,
			expectedFilePath: filepath.Join(dirName, validDirNameWithDaprYAMLFile, "dapr.yaml"),
		},
		{
			name:             "valid directory path with no dapr.yaml file",
			input:            filepath.Join(dirName, validDirWithNoDaprYAML),
			expectedErr:      true,
			expectedFilePath: "",
		},
		{
			name:             "non existent dir",
			input:            nonExistentDirName,
			expectedErr:      true,
			expectedFilePath: "",
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			filePath, err := FindFileInDir(tc.input, "dapr.yaml")
			assert.Equal(t, tc.expectedErr, err != nil)
			assert.Equal(t, tc.expectedFilePath, filePath)
		})
	}
}

func TestPrintDetail(t *testing.T) {
	type fooStruct struct {
		Field1 string
		Field2 int
	}

	testcases := []struct {
		name        string
		format      string
		list        interface{}
		expected    string
		shouldError bool
	}{
		{
			name:        "multiple items in JSON format",
			format:      "json",
			list:        []fooStruct{{Field1: "test1", Field2: 1}, {Field1: "test2", Field2: 2}},
			expected:    "[\n  {\n    \"Field1\": \"test1\",\n    \"Field2\": 1\n  },\n  {\n    \"Field1\": \"test2\",\n    \"Field2\": 2\n  }\n]",
			shouldError: false,
		},
		{
			name:        "single item in JSON format",
			format:      "json",
			list:        []fooStruct{{Field1: "test1", Field2: 1}},
			expected:    "[\n  {\n    \"Field1\": \"test1\",\n    \"Field2\": 1\n  }\n]",
			shouldError: false,
		},
		{
			name:        "no items in JSON format",
			format:      "json",
			list:        []fooStruct{},
			expected:    "[]",
			shouldError: false,
		},
		{
			name:        "multiple items in YAML format",
			format:      "yaml",
			list:        []fooStruct{{Field1: "test1", Field2: 1}, {Field1: "test2", Field2: 2}},
			expected:    "- field1: test1\n  field2: 1\n- field1: test2\n  field2: 2\n",
			shouldError: false,
		},
		{
			name:        "single item in YAML format",
			format:      "yaml",
			list:        []fooStruct{{Field1: "test1", Field2: 1}},
			expected:    "- field1: test1\n  field2: 1\n",
			shouldError: false,
		},
		{
			name:        "no items in YAML format",
			format:      "yaml",
			list:        []fooStruct{},
			expected:    "[]\n",
			shouldError: false,
		},
		{
			name:        "invalid format",
			format:      "invalid",
			list:        []fooStruct{},
			expected:    "",
			shouldError: true,
		},
		{
			name:        "invalid JSON",
			format:      "json",
			list:        math.Inf(1),
			expected:    "",
			shouldError: true,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := PrintDetail(&buf, tc.format, tc.list)
			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, buf.String())
			}
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

func TestSanitizeDir(t *testing.T) {
	testcases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "directory with single quote in three places",
			input:    "C:\\Use'rs\\sta'rk\\Docum'ents",
			expected: "C:\\Use''rs\\sta''rk\\Docum''ents",
		},
		{
			name:     "directory with single quote in two places",
			input:    "C:\\Use'rs\\sta'rk",
			expected: "C:\\Use''rs\\sta''rk",
		},
		{
			name:     "directory with single quote in username",
			input:    "C:\\Users\\Debash'ish",
			expected: "C:\\Users\\Debash''ish",
		},
		{
			name:     "directory with no single quote",
			input:    "C:\\Users\\Shubham",
			expected: "C:\\Users\\Shubham",
		},
		{
			name:     "directory with single quote in one place",
			input:    "C:\\Use'rs\\Shubham",
			expected: "C:\\Use''rs\\Shubham",
		},
		{
			name:     "directory with single quote in many places in username",
			input:    "C:\\Users\\Shu'bh'am",
			expected: "C:\\Users\\Shu''bh''am",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := SanitizeDir(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
