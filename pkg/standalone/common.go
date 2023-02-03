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
	"os"
	path_filepath "path/filepath"
	"runtime"
	"strings"
)

const (
	DefaultDaprDirName      = ".dapr"
	DefaultConfigFileName   = "config.yaml"
	DefaultResourcesDirName = "resources"

	defaultDaprBinDirName    = "bin"
	defaultComponentsDirName = "components"
)

// GetDaprRuntimePath returns the dapr runtime installation path.
// The order of precedence is:
//  1. From --runtime-path command line flag appended with `.dapr`
//  2. From DAPR_RUNTIME_PATH environment variable appended with `.dapr`
//  3. default $HOME/.dapr
func GetDaprRuntimePath(inputInstallPath string) (string, error) {
	installPath := strings.TrimSpace(inputInstallPath)
	if installPath != "" {
		return path_filepath.Join(installPath, DefaultDaprDirName), nil
	}

	envDaprDir := strings.TrimSpace(os.Getenv("DAPR_RUNTIME_PATH"))
	if envDaprDir != "" {
		return path_filepath.Join(envDaprDir, DefaultDaprDirName), nil
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
	return path_filepath.Join(daprDir, defaultComponentsDirName)
}

func GetDaprConfigPath(daprDir string) string {
	return path_filepath.Join(daprDir, DefaultConfigFileName)
}
