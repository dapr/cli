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

package standalone

import (
	"fmt"
	"os"
	path_filepath "path/filepath"
	"runtime"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

const (
	defaultDaprDirName    = ".dapr"
	defaultDaprBinDirName = "bin"
	defaultConfigFileName = "config.yaml"
)

func defaultDaprDirPath() string {
	homeDir, _ := os.UserHomeDir()
	return path_filepath.Join(homeDir, defaultDaprDirName)
}

func defaultDaprBinPath() string {
	return path_filepath.Join(defaultDaprDirPath(), defaultDaprBinDirName)
}

func binaryFilePath(binaryDir string, binaryFilePrefix string) string {
	binaryPath := path_filepath.Join(binaryDir, binaryFilePrefix)
	if runtime.GOOS == daprWindowsOS {
		binaryPath += ".exe"
	}
	return binaryPath
}

func DefaultComponentsDirPath() string {
	return path_filepath.Join(defaultDaprDirPath(), utils.DefaultComponentsDirName)
}

func DefaultResourcesDirPath() string {
	return path_filepath.Join(defaultDaprDirPath(), utils.DefaultResourcesDirName)
}

// GetResourcesDir returns the path to the resources directory if it exists, otherwise it returns the path of components directory.
// TODO: Remove this function and replace all its usage with above defined `DefaultResourcesDirPath` when `--components-path` flag is removed.
func GetResourcesDir() string {
	defaultResourcesDirPath := DefaultResourcesDirPath()
	if _, err := os.Stat(defaultResourcesDirPath); os.IsNotExist(err) {
		return DefaultComponentsDirPath()
	}
	return defaultResourcesDirPath
}

func DefaultConfigFilePath() string {
	return path_filepath.Join(defaultDaprDirPath(), defaultConfigFileName)
}

// copyFilesAndCreateSymlink copies files from src to dest. It deletes the existing files in dest before copying from src.
// this method also deletes the components dir and makes it as a symlink to resources directory.
// please see this comment for more details:https://github.com/dapr/cli/pull/1149#issuecomment-1364424345
// TODO: Remove this function when `--components-path` flag is removed.
func copyFilesAndCreateSymlink(src, dest string) error {
	var err error
	if _, err = os.Stat(src); err != nil {
		// if the src directory does not exist, create symlink and return nil, because there is nothing to copy from.
		if os.IsNotExist(err) {
			err = createSymLink(dest, src)
			if err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("error reading directory %s: %w", src, err)
	}
	if err = moveDir(src, dest); err != nil {
		return err
	}
	if err = createSymLink(dest, src); err != nil {
		return err
	}
	return nil
}

// moveDir moves files from src to dest. If there are files in src, it deletes the existing files in dest before copying from src.
func moveDir(src, dest string) error {
	destFiles, err := os.ReadDir(dest)
	if err != nil {
		return fmt.Errorf("error reading files from %s: %w", dest, err)
	}
	srcFiles, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("error reading files from %s: %w", src, err)
	}
	if len(srcFiles) > 0 {
		// delete the existing files in dest before copying from src iff there are files in src.
		for _, file := range destFiles {
			err = os.Remove(path_filepath.Join(dest, file.Name()))
			if err != nil {
				return fmt.Errorf("error removing file %s: %w", file.Name(), err)
			}
		}
		print.InfoStatusEvent(os.Stdout, "Moving files from %q to %q", src, dest)
		var content []byte
		for _, file := range srcFiles {
			content, err = os.ReadFile(path_filepath.Join(src, file.Name()))
			if err != nil {
				return fmt.Errorf("error reading file %s: %w", file.Name(), err)
			}
			// #nosec G306
			err = os.WriteFile(path_filepath.Join(dest, file.Name()), content, 0o644)
			if err != nil {
				return fmt.Errorf("error writing file %s: %w", file.Name(), err)
			}
		}
	}
	// delete the components dir and make it as a symlink to resources directory.
	err = os.RemoveAll(src)
	if err != nil {
		return fmt.Errorf("error removing directory %s: %w", src, err)
	}
	return nil
}

// createSymLink creates a symlink from dirName to symLink.
func createSymLink(dirName, symLink string) error {
	if _, err := os.Stat(dirName); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory %s does not exist", dirName)
		}
		return fmt.Errorf("error reading directory %s: %w", dirName, err)
	}
	err := os.Symlink(dirName, symLink)
	if err != nil {
		return fmt.Errorf("error creating symlink from %s to %s: %w", dirName, symLink, err)
	}
	return nil
}
