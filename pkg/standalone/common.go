// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"os"
	path_filepath "path/filepath"
	"runtime"
)

const (
	defaultDaprDirName       = ".dapr"
	defaultComponentsDirName = "components"
	defaultConfigFileName    = "config.yaml"
)

func defaultDaprDirPath() string {
	homeDir, _ := os.UserHomeDir()
	return path_filepath.Join(homeDir, defaultDaprDirName)
}

func binaryFilePath(binaryDir string, binaryFilePrefix string) string {
	binaryPath := path_filepath.Join(binaryDir, binaryFilePrefix)
	if runtime.GOOS == daprWindowsOS {
		binaryPath += ".exe"
	}
	return binaryPath
}

func DefaultComponentsDirPath() string {
	return path_filepath.Join(defaultDaprDirPath(), defaultComponentsDirName)
}

func DefaultConfigFilePath() string {
	return path_filepath.Join(defaultDaprDirPath(), defaultConfigFileName)
}
