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

package standalone

import (
	"os"
	path_filepath "path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateSymLink(t *testing.T) {
	// create a temp dir to hold the symlink and actual directory.
	tempDir := createTempDir(t, "dapr-test", "")
	defer cleanupTempDir(t, tempDir)
	rsrcDir := createTempDir(t, "resources", tempDir)
	existingSymLinkDir := createTempDir(t, "components_exist", tempDir)
	tests := []struct {
		name          string
		actualDirName string
		symLinkName   string
		expectedError bool
	}{
		{
			name:          "create symlink for resources directory",
			actualDirName: rsrcDir,
			symLinkName:   path_filepath.Join(tempDir, "components"),
			expectedError: false,
		},
		{
			name:          "create symlink when resources directory does not exist",
			actualDirName: "invalid-dir",
			symLinkName:   "components",
			expectedError: true,
		},
		{
			name:          "create symlink when symlink named directory already exists",
			actualDirName: rsrcDir,
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

func TestCopyFilesAndCreateSymlink(t *testing.T) {
	// create a temp dir to hold the symlink and actual directory.
	tempDir := createTempDir(t, "dapr-test", "")
	defer cleanupTempDir(t, tempDir)
	rsrcDir := createTempDir(t, "resources", tempDir)
	cmptDir := createTempDir(t, "components", tempDir)
	cmptFile := createTempFile(t, cmptDir, "pubsub.yaml")
	rsrcFile := createTempFile(t, cmptDir, "pubsub-rsrc.yaml")
	tests := []struct {
		name            string
		actualDirName   string
		symLinkName     string
		expectedError   bool
		presentFileName string
	}{
		{
			name:            "copy files and create symlink for resources directory when components dir exists",
			actualDirName:   rsrcDir,
			symLinkName:     cmptDir,
			expectedError:   false,
			presentFileName: cmptFile,
		},
		{
			name:            "copy files and create symlink for resources directory when components dir does not exists",
			actualDirName:   rsrcDir,
			symLinkName:     path_filepath.Join(tempDir, "components-not-exist"),
			expectedError:   false,
			presentFileName: rsrcFile,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := copyFilesAndCreateSymlink(tt.symLinkName, tt.actualDirName)
			assert.Equal(t, tt.expectedError, err != nil)
			// check if the files are copied.
			assert.FileExists(t, path_filepath.Join(tt.symLinkName, path_filepath.Base(tt.presentFileName)))
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
