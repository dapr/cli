// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"bufio"
	"os/exec"
	"strings"
)

// Values for these are injected by the build.
var (
	gitcommit, gitversion string
)

// GetRuntimeVersion returns the version for the local Dapr runtime.
func GetRuntimeVersion() string {
	daprBinDir := defaultDaprBinPath()
	daprCMD := binaryFilePath(daprBinDir, "daprd")

	out, err := exec.Command(daprCMD, "--version").Output()
	if err != nil {
		return "n/a\n"
	}
	return string(out)
}

// GetDashboardVersion returns the version for the local Dapr dashboard.
func GetDashboardVersion() string {
	daprBinDir := defaultDaprBinPath()
	dashboardCMD := binaryFilePath(daprBinDir, "dashboard")

	out, err := exec.Command(dashboardCMD, "--version").Output()
	if err != nil {
		return "n/a\n"
	}
	return string(out)
}

// GetBuildInfo returns build info for the CLI and the local Dapr runtime.
func GetBuildInfo(version string) string {
	daprBinDir := defaultDaprBinPath()
	daprCMD := binaryFilePath(daprBinDir, "daprd")

	strs := []string{
		"CLI:",
		"\tVersion: " + version,
		"\tGit Commit: " + gitcommit,
		"\tGit Version: " + gitversion,
		"Runtime:",
	}

	out, err := exec.Command(daprCMD, "--build-info").Output()
	if err != nil {
		// try '--version' for older runtime version
		out, err = exec.Command(daprCMD, "--version").Output()
	}
	if err != nil {
		strs = append(strs, "\tN/A")
	} else {
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			strs = append(strs, "\t"+scanner.Text())
		}
	}
	return strings.Join(strs, "\n")
}
