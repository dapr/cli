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
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectInstallMethod_Homebrew(t *testing.T) {
	assert.Equal(t, InstallHomebrew, DetectInstallMethod("/opt/homebrew/Cellar/dapr-cli/1.18.3/bin/dapr"))
}

func TestDetectInstallMethod_Direct(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("depends on the Windows registry")
	}
	assert.Equal(t, InstallDirect, DetectInstallMethod("/usr/local/bin/dapr"))
}

func TestPackageManagerHint(t *testing.T) {
	const brew = "brew upgrade dapr/tap/dapr-cli"
	tests := []struct {
		name      string
		method    InstallMethod
		requested string
		current   string
		want      string
	}{
		{"homebrew latest", InstallHomebrew, "", "1.18.3",
			"The Dapr CLI was installed with Homebrew. To update it, run:\n  " + brew},
		{"homebrew version", InstallHomebrew, "1.17.0", "1.18.3",
			"The Dapr CLI was installed with Homebrew, which can't install a specific version. To update to the latest version, run:\n  " + brew},
		{"msi latest", InstallMSI, "", "1.18.3",
			"The Dapr CLI was installed with the Windows installer (winget). To update it, run:\n  winget upgrade Dapr.CLI"},
		{"msi version upgrade", InstallMSI, "1.19.0", "1.18.3",
			"The Dapr CLI was installed with the Windows installer (winget). To install version 1.19.0, run:\n  winget install Dapr.CLI --version 1.19.0"},
		{"msi rc", InstallMSI, "v1.19.0-rc.1", "1.18.3",
			"The Dapr CLI was installed with the Windows installer (winget). To install version 1.19.0-rc.1, run:\n  winget install Dapr.CLI.Preview --version 1.19.0-rc.1"},
		{"msi downgrade", InstallMSI, "1.17.0", "1.18.3",
			"The Dapr CLI was installed with the Windows installer (winget). To install version 1.17.0, run:\n  winget install Dapr.CLI --version 1.17.0\nThe installer doesn't allow downgrades, so first run:\n  winget uninstall Dapr.CLI"},
		{"msi non-semver current", InstallMSI, "1.17.0", "edge",
			"The Dapr CLI was installed with the Windows installer (winget). To install version 1.17.0, run:\n  winget install Dapr.CLI --version 1.17.0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, PackageManagerHint(tc.method, tc.requested, tc.current))
		})
	}
}
