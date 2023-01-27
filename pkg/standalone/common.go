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
)

const (
	DefaultDaprDirName      = ".dapr"
	DefaultConfigFileName   = "config.yaml"
	DefaultResourcesDirName = "resources"

	defaultDaprBinDirName    = "bin"
	defaultComponentsDirName = "components"
)

// GetDaprPath returns the dapr installation path.
// The order of precedence is:
//  1. From --dapr-path command line flag
//  2. From DAPR_PATH environment variable
//  3. $HOME/.dapr
func GetDaprPath(inputInstallPath string) (string, error) {
	if inputInstallPath != "" {
		return inputInstallPath, nil
	}

	envDaprDir := os.Getenv("DAPR_PATH")
	if envDaprDir != "" {
		return envDaprDir, nil
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
	daprPath, err := GetDaprPath(inputInstallPath)
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
