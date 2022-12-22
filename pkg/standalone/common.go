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
	defaultDaprDirName       = ".dapr"
	defaultDaprBinDirName    = "bin"
	defaultComponentsDirName = "components"
	defaultConfigFileName    = "config.yaml"
)

// GetDaprDirPath - return the dapr installation path to employ. In order of
// precednce:
//  1. if present --dapr-path command line flag specified by the user
//  2. if present DAPR_PATH environment variable
//  3. $HOME/.dapr
func GetDaprDirPath(inputInstallPath string) (string, error) {
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

	return path_filepath.Join(homeDir, defaultDaprDirName), nil
}

func daprBinPath(daprDir string) string {
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
	daprPath, err := GetDaprDirPath(inputInstallPath)
	if err != nil {
		return "", err
	}

	return binaryFilePathWithDir(daprBinPath(daprPath), binaryFilePrefix), nil
}

func DaprComponentsPath(daprDir string) string {
	return path_filepath.Join(daprDir, defaultComponentsDirName)
}

func DaprConfigPath(daprDir string) string {
	return path_filepath.Join(daprDir, defaultConfigFileName)
}
