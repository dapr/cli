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
	"strings"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

const (
	DefaultDaprDirName      = ".dapr"
	DefaultConfigFileName   = "config.yaml"
	DefaultResourcesDirName = "resources"

	defaultDaprBinDirName = "bin"
)

// GetDaprRuntimePath returns the dapr runtime installation path.
// daprRuntimePath is based on the --runtime-path command line flag.
// The order of precedence is:
//  1. From --runtime-path command line flag appended with `.dapr`
//  2. From DAPR_RUNTIME_PATH environment variable appended with `.dapr`
//  3. default $HOME/.dapr
func GetDaprRuntimePath(daprRuntimePath string) (string, error) {
	runtimePath := strings.TrimSpace(daprRuntimePath)
	if runtimePath != "" {
		return path_filepath.Join(runtimePath, DefaultDaprDirName), nil
	}

	envRuntimePath := strings.TrimSpace(os.Getenv("DAPR_RUNTIME_PATH"))
	if envRuntimePath != "" {
		return path_filepath.Join(envRuntimePath, DefaultDaprDirName), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path_filepath.Join(homeDir, DefaultDaprDirName), nil
}

func getDaprBinPath(daprDir string) string {
	return path_filepath.Join(daprDir, defaultDaprBinDirName)
}

func binaryFilePathWithDir(binaryDir string, binaryFilePrefix string) string {
	binaryPath := path_filepath.Join(binaryDir, binaryFilePrefix)
	if runtime.GOOS == daprWindowsOS {
		binaryPath += ".exe"
	}
	return binaryPath
}

func lookupBinaryFilePath(inputInstallPath string, binaryFilePrefix string) (string, error) {
	daprPath, err := GetDaprRuntimePath(inputInstallPath)
	if err != nil {
		return "", err
	}

	return binaryFilePathWithDir(getDaprBinPath(daprPath), binaryFilePrefix), nil
}

func GetDaprComponentsPath(daprDir string) string {
	return path_filepath.Join(daprDir, utils.DefaultComponentsDirName)
}

func GetDaprResourcesPath(daprDir string) string {
	return path_filepath.Join(daprDir, utils.DefaultResourcesDirName)
}

func GetDaprConfigPath(daprDir string) string {
	return path_filepath.Join(daprDir, DefaultConfigFileName)
}

// GetResourcesDir returns the path to the resources directory if it exists, otherwise it returns the path of components directory.
// TODO: Remove this function and replace all its usage with above defined `DefaultResourcesDirPath` when `--components-path` flag is removed.
func GetResourcesDir(daprDir string) string {
	defaultResourcesDirPath := GetDaprResourcesPath(daprDir)
	if _, err := os.Stat(defaultResourcesDirPath); os.IsNotExist(err) {
		return GetDaprComponentsPath(daprDir)
	}
	return defaultResourcesDirPath
}

// moveDir moves files from src to dest. If there are files in src, it deletes the existing files in dest before copying from src and then deletes the src directory.
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
		// delete the existing files in dest before copying from src if there are files in src.
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
