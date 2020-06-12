package standalone

import (
	"os"
	path_filepath "path/filepath"
	"runtime"
)

const (
	defaultDaprDirName       = ".dapr"
	DefaultComponentsDirName = "components"
	defaultConfigDirName     = "config"
)

func getHomeFolder() string {
	homePath := os.Getenv("HOME")
	if runtime.GOOS == daprWindowsOS {
		homePath = os.Getenv("USERPROFILE")
	}
	return homePath
}

// GetDefaultFolderPath returns the default requested path to the requested path
func GetDefaultFolderPath(dirName string) string {
	homePath := getHomeFolder()
	if dirName == defaultDaprDirName {
		return path_filepath.Join(homePath, defaultDaprDirName)
	}
	return path_filepath.Join(homePath, defaultDaprDirName, dirName)
}
