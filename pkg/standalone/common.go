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

func homeFolder() string {
	homePath := os.Getenv("HOME")
	if runtime.GOOS == daprWindowsOS {
		homePath = os.Getenv("USERPROFILE")
	}
	return homePath
}

// DefaultFolderPath returns the default requested path to the requested path
func defaultFolderPath(dirName string) string {
	homePath := homeFolder()
	if dirName == defaultDaprDirName {
		return path_filepath.Join(homePath, defaultDaprDirName)
	}
	return path_filepath.Join(homePath, defaultDaprDirName, dirName)
}

func binaryInstallationPath(installLocation string) string {
	if installLocation != "" {
		return installLocation
	}
	if runtime.GOOS == daprWindowsOS {
		return daprDefaultWindowsInstallPath
	}
	return daprDefaultLinuxAndMacInstallPath
}

func daprdBinaryFilePath(installLocation string) string {
	destDir := binaryInstallationPath(installLocation)
	daprdBinaryPath := path_filepath.Join(destDir, daprRuntimeFilePrefix)
	if runtime.GOOS == daprWindowsOS {
		daprdBinaryPath = path_filepath.Join(daprdBinaryPath, ".exe")
	}
	return daprdBinaryPath
}

func DefaultComponentsDirPath() string {
	return defaultFolderPath(defaultComponentsDirName)
}

func DefaultConfigFilePath() string {
	configPath := defaultFolderPath(defaultDaprDirName)
	filePath := path_filepath.Join(configPath, defaultConfigFileName)
	return filePath
}
