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
	"fmt"
	"github.com/dapr/cli/pkg/print"
	cli_ver "github.com/dapr/cli/pkg/version"
	"io"
	"net/http"
	"os"
	"path"
	path_filepath "path/filepath"
	"runtime"
	"strings"
)

const (
	defaultDaprDirName       = ".dapr"
	defaultDaprBinDirName    = "bin"
	defaultComponentsDirName = "components"
	defaultConfigFileName    = "config.yaml"
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
	return path_filepath.Join(defaultDaprDirPath(), defaultComponentsDirName)
}

func DefaultConfigFilePath() string {
	return path_filepath.Join(defaultDaprDirPath(), defaultConfigFileName)
}

func findDashboardVersion(ver string) string {
	if isEmbedded {
		return dashboardVersion
	}

	if ver == latestVersion {
		v, err := cli_ver.GetDashboardVersion()
		if err != nil {
			print.WarningStatusEvent(os.Stdout, "cannot get the latest dashboard version: '%s'. Try specifying --dashboard-version=<desired_version>", err)
			print.WarningStatusEvent(os.Stdout, "continuing, but dashboard will be unavailable")
		}
		return v
	}

	return ver
}

func findRuntimeVersion(ver string) (string, error) {
	if isEmbedded {
		return runtimeVersion, nil
	}

	if ver == latestVersion {
		v, err := cli_ver.GetDaprVersion()
		if err != nil {
			return "", fmt.Errorf("cannot get the latest release version: '%s'. Try specifying --runtime-version=<desired_version>", err)
		}
		return v, nil
	}
	return ver, nil
}

func prepareDaprInstallDir(daprBinDir string) error {
	err := os.MkdirAll(daprBinDir, 0777)
	if err != nil {
		return err
	}

	err = os.Chmod(daprBinDir, 0777)
	if err != nil {
		return err
	}

	return nil
}

func archiveExt() string {
	ext := "tar.gz"
	if runtime.GOOS == daprWindowsOS {
		ext = "zip"
	}

	return ext
}

func downloadBinary(dir, version, binaryFilePrefix, githubRepo string) (string, error) {
	fileURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/v%s/%s",
		cli_ver.DaprGitHubOrg,
		githubRepo,
		version,
		binaryName(binaryFilePrefix))

	return downloadFile(dir, fileURL)
}

func binaryName(binaryFilePrefix string) string {
	return fmt.Sprintf("%s_%s_%s.%s", binaryFilePrefix, runtime.GOOS, runtime.GOARCH, archiveExt())
}

// nolint:gosec
func downloadFile(dir string, url string) (string, error) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]

	filepath := path.Join(dir, fileName)
	_, err := os.Stat(filepath)
	if os.IsExist(err) {
		return "", nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("version not found from url: %s", url)
	} else if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed with %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return filepath, nil
}
