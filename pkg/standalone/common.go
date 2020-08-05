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
	defaultDaprBinDirName    = "bin"
	defaultComponentsDirName = "components"
	defaultConfigFileName    = "config.yaml"
)

// DefaultDaprDirPath returns the default Dapr install path
func DefaultDaprDirPath() string {
	homeDir, _ := os.UserHomeDir()
	return path_filepath.Join(homeDir, defaultDaprDirName)
}

// DefaultDaprBinPath returns the default install path for Dapr binaries
func DefaultDaprBinPath() string {
	return path_filepath.Join(DefaultDaprDirPath(), defaultDaprBinDirName)
}

// BinaryFilePath returns the OS-specific default binary file path
func BinaryFilePath(binaryDir string, binaryFilePrefix string) string {
	binaryPath := path_filepath.Join(binaryDir, binaryFilePrefix)
	if runtime.GOOS == daprWindowsOS {
		binaryPath += ".exe"
	}
	return binaryPath
}

// DefaultComponentsDirPath returns the default path for the Dapr components directory
func DefaultComponentsDirPath() string {
	return path_filepath.Join(DefaultDaprDirPath(), defaultComponentsDirName)
}

// DefaultConfigFilePath returns the default path for the Dapr configuration file
func DefaultConfigFilePath() string {
	return path_filepath.Join(DefaultDaprDirPath(), defaultConfigFileName)
}
