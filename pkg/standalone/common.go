package standalone

import (
	"os"
	path_filepath "path/filepath"
	"runtime"
)

// getDefaultComponentsFolder returns the hidden .components folder created at init time
func getDefaultComponentsFolder() string {
	const daprDirName = ".dapr"
	const componentsDirName = "components"
	daprDirPath := os.Getenv("HOME")
	if runtime.GOOS == daprWindowsOS {
		daprDirPath = os.Getenv("USERPROFILE")
	}

	defaultComponentsPath := path_filepath.Join(daprDirPath, daprDirName, componentsDirName)
	return defaultComponentsPath
}
