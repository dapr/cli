/*
Copyright 2026 The Dapr Authors
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

package selfupdate

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// InstallMethod describes how the running CLI was installed.
type InstallMethod int

const (
	// InstallDirect is a plain binary install that the CLI can replace itself.
	InstallDirect InstallMethod = iota
	// InstallHomebrew is an install managed by Homebrew.
	InstallHomebrew
	// InstallMSI is an install managed by the Windows installer.
	InstallMSI
)

// msiInstalledFunc is a variable so tests can simulate an MSI install.
var msiInstalledFunc = msiInstalled

// DetectInstallMethod determines how the CLI at resolvedExePath was installed.
func DetectInstallMethod(resolvedExePath string) InstallMethod {
	if strings.Contains(strings.ReplaceAll(resolvedExePath, "\\", "/"), "/Cellar/") {
		return InstallHomebrew
	}
	if msiInstalledFunc() {
		return InstallMSI
	}
	return InstallDirect
}

// PackageManagerHint returns the instructions for updating a package-manager
// managed install, or an empty string for InstallDirect.
func PackageManagerHint(m InstallMethod, requestedVersion, currentVersion string) string {
	switch m {
	case InstallHomebrew:
		if requestedVersion == "" {
			return "The Dapr CLI was installed with Homebrew. To update it, run:\n  brew upgrade dapr/tap/dapr-cli"
		}
		return "The Dapr CLI was installed with Homebrew, which can't install a specific version. To update to the latest version, run:\n  brew upgrade dapr/tap/dapr-cli"
	case InstallMSI:
		if requestedVersion == "" {
			return "The Dapr CLI was installed with the Windows installer (winget). To update it, run:\n  winget upgrade Dapr.CLI"
		}
		v := NormalizeVersion(requestedVersion)
		id := "Dapr.CLI"
		if strings.Contains(v, "-rc.") {
			id = "Dapr.CLI.Preview"
		}
		hint := fmt.Sprintf("The Dapr CLI was installed with the Windows installer (winget). To install version %s, run:\n  winget install %s --version %s", v, id, v)
		if isDowngrade(v, currentVersion) {
			hint += "\nThe installer doesn't allow downgrades, so first run:\n  winget uninstall Dapr.CLI"
		}
		return hint
	}
	return ""
}

func isDowngrade(target, current string) bool {
	t, err := semver.NewVersion(target)
	if err != nil {
		return false
	}
	c, err := semver.NewVersion(NormalizeVersion(current))
	if err != nil {
		return false
	}
	return t.LessThan(c)
}
