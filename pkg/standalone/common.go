package standalone

import (
	"os"
	path_filepath "path/filepath"
	"runtime"
)

// getDefaultComponentsFolder returns the hidden .components folder created under install directory at init time
func getDefaultComponentsFolder() string {
	const daprDirName = ".dapr"
	const componentsDirName = "components"
	daprDirPath := os.Getenv("HOME")
	if runtime.GOOS == daprWindowsOS {
		daprDirPath = path_filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"))
	}

	defaultComponentsPath := path_filepath.Join(daprDirPath, daprDirName, componentsDirName)
	return defaultComponentsPath
}
