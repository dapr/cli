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

	"github.com/dapr/cli/pkg/print"
)

const (
	DefaultComponentsDirName = "components"
	DefaultResourcesDirName  = "resources"

	defaultDaprDirName    = ".dapr"
	defaultDaprBinDirName = "bin"
	defaultConfigFileName = "config.yaml"
)

func defaultDaprDirPath() string {
	homeDir, _ := os.UserHomeDir()
	return path_filepath.Join(homeDir, defaultDaprDirName)
}

func defaultDaprBinPath() string {
	return path_filepath.Join(defaultDaprDirPath(), defaultDaprBinDirName)
}

func binaryFilePath(binaryDir string, binaryFilePrefix string) string {
	binaryPath := path_filepath.Join(binaryDir, binaryFilePrefix)
	if runtime.GOOS == daprWindowsOS {
		binaryPath += ".exe"
	}
	return binaryPath
}

func DefaultComponentsDirPath() string {
	return path_filepath.Join(defaultDaprDirPath(), DefaultComponentsDirName)
}

func DefaultResourcesDirPath() string {
	return path_filepath.Join(defaultDaprDirPath(), DefaultResourcesDirName)
}

// DefaultResourcesDirPrecedence returns the path to the resources directory if it exists, or otherwise the components directory.
// TODO: Remove this function and use `DefaultResourcesDirPath` when `--components-path` flag is removed.
func DefaultResourcesDirPrecedence() string {
	defaultResourcesDirPath := DefaultResourcesDirPath()
	if _, err := os.Stat(defaultResourcesDirPath); os.IsNotExist(err) {
		return DefaultComponentsDirPath()
	}
	return defaultResourcesDirPath
}

func DefaultConfigFilePath() string {
	return path_filepath.Join(defaultDaprDirPath(), defaultConfigFileName)
}

// emptyAndCopyFiles copies files from src to dest. It deletes the existing files in dest before copying from src.
// TODO: Remove this function when `--components-path` flag is removed.
func emptyAndCopyFiles(src, dest string) error {
	if _, err := os.Stat(src); err != nil {
		// if the src directory does not exist, return nil, because there is nothing to copy from.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	files, err := os.ReadDir(dest)
	if err != nil {
		return err
	}
	for _, file := range files {
		err = os.Remove(dest + "/" + file.Name())
		if err != nil {
			return err
		}
	}
	files, err = os.ReadDir(src)
	if err != nil {
		return err
	}
	if len(files) > 0 {
		print.InfoStatusEvent(os.Stdout, "Moving files from %q to %q", src, dest)
		for _, file := range files {
			content, err := os.ReadFile(src + "/" + file.Name())
			if err != nil {
				return err
			}
			// #nosec G306
			err = os.WriteFile(dest+"/"+file.Name(), content, 0o644)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
