/*
Copyright 2023 The Dapr Authors
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

package standalone

import (
	"os"
	path_filepath "path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDaprPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "error getting home dir")

	t.Run("without flag value or env var", func(t *testing.T) {
		p, err := GetDaprRuntimePath("")
		require.NoError(t, err)
		assert.Equal(t, p, path_filepath.Join(homeDir, DefaultDaprDirName), "path should be $HOME/.dapr")
	})

	t.Run("check trim spaces", func(t *testing.T) {
		p, err := GetDaprRuntimePath("      ")
		require.NoError(t, err)
		assert.Equal(t, path_filepath.Join(homeDir, DefaultDaprDirName), p, "path should be $HOME/.dapr")

		t.Setenv("DAPR_RUNTIME_PATH", "      ")
		p, err = GetDaprRuntimePath("")
		require.NoError(t, err)
		assert.Equal(t, path_filepath.Join(homeDir, DefaultDaprDirName), p, "path should be $HOME/.dapr")
	})

	t.Run("with flag value", func(t *testing.T) {
		input := path_filepath.Join("path", "to", "dapr")
		p, err := GetDaprRuntimePath(input)
		require.NoError(t, err)
		assert.Equal(t, path_filepath.Join(input, ".dapr"), p, "path should be /path/to/dapr/.dapr")
	})

	t.Run("with env var", func(t *testing.T) {
		input := path_filepath.Join("path", "to", "dapr")
		t.Setenv("DAPR_RUNTIME_PATH", input)
		p, err := GetDaprRuntimePath("")
		require.NoError(t, err)
		assert.Equal(t, path_filepath.Join(input, ".dapr"), p, "path should be /path/to/dapr/.dapr")
	})

	t.Run("with flag value and env var", func(t *testing.T) {
		input := path_filepath.Join("path", "to", "dapr")
		input2 := path_filepath.Join("path", "to", "dapr2")
		t.Setenv("DAPR_RUNTIME_PATH", input2)
		p, err := GetDaprRuntimePath(input)
		require.NoError(t, err)
		assert.Equal(t, path_filepath.Join(input, ".dapr"), p, "path should be /path/to/dapr/.dapr")
	})
}

func TestCreateSymLink(t *testing.T) {
	// create a temp dir to hold the symlink and actual directory.
	tempDir := createTempDir(t, "dapr-test", "")
	defer cleanupTempDir(t, tempDir)
	originalDir := createTempDir(t, "original_dir", tempDir)
	existingSymLinkDir := createTempDir(t, "new_name_exist", tempDir)
	tests := []struct {
		name          string
		actualDirName string
		symLinkName   string
		expectedError bool
	}{
		{
			name:          "create symlink for resources directory",
			actualDirName: originalDir,
			symLinkName:   path_filepath.Join(tempDir, "new_name"),
			expectedError: false,
		},
		{
			name:          "create symlink when resources directory does not exist",
			actualDirName: "invalid-dir",
			symLinkName:   "new_name",
			expectedError: true,
		},
		{
			name:          "create symlink when symlink named directory already exists",
			actualDirName: originalDir,
			symLinkName:   existingSymLinkDir,
			expectedError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := createSymLink(tt.actualDirName, tt.symLinkName)
			assert.Equal(t, tt.expectedError, err != nil)
		})
	}
}

func TestMoveDir(t *testing.T) {
	// create a temp dir to hold the source and destination directory.
	tempDir := createTempDir(t, "dapr-test", "")
	defer cleanupTempDir(t, tempDir)
	// create a file in the source and destination directory.
	src1 := createTempDir(t, "src1", tempDir)
	dest2 := createTempDir(t, "dest2", tempDir)
	srcFile := createTempFile(t, src1, "pubsub.yaml")
	destFile := createTempFile(t, dest2, "pubsub-dest.yaml")
	tests := []struct {
		name            string
		srcDirName      string
		destDirName     string
		expectedError   bool
		presentFileName string
	}{
		{
			name:            "move directory when source directory contains files",
			srcDirName:      src1,
			destDirName:     createTempDir(t, "dest1", tempDir),
			expectedError:   false,
			presentFileName: path_filepath.Base(srcFile),
		},
		{
			name:            "move directory when source directory does not contain files",
			srcDirName:      createTempDir(t, "src2", tempDir),
			destDirName:     dest2,
			expectedError:   false,
			presentFileName: path_filepath.Base(destFile),
		},
		{
			name:          "move directory when source directory does not exists",
			srcDirName:    path_filepath.Join(tempDir, "non-existent-source-dir"),
			destDirName:   createTempDir(t, "dest3", tempDir),
			expectedError: true,
		},
		{
			name:          "move directory when destination directory does not exists",
			srcDirName:    createTempDir(t, "src4", tempDir),
			destDirName:   path_filepath.Join(tempDir, "non-existent-dir-dir"),
			expectedError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := moveDir(tt.srcDirName, tt.destDirName)
			assert.Equal(t, tt.expectedError, err != nil)
			if tt.presentFileName != "" {
				// check if the files are moved correctly.
				assert.FileExists(t, path_filepath.Join(tt.destDirName, tt.presentFileName))
			}
		})
	}
}

func createTempDir(t *testing.T, tempDirName, containerDir string) string {
	dirName, err := os.MkdirTemp(containerDir, tempDirName)
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
