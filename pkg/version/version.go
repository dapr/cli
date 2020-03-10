// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package version

import (
	"os/exec"
	"regexp"
	"runtime"
)

// GetRuntimeVersion returns the version for the local Dapr runtime.
func GetRuntimeVersion() string {
	runtimeName := ""
	if runtime.GOOS == "windows" {
		runtimeName = "daprd.exe"
	} else {
		runtimeName = "daprd"
	}

	out, err := exec.Command(runtimeName, "--version").Output()
	if err != nil {
		return "n/a\n"
	}
	return string(out)

}

// GetCLIVersion returns the version for the Dapr cli.
func GetCLIVersion() string {
	cliName := ""
	if runtime.GOOS == "windows" {
		cliName = "dapr.exe"
	} else {
		cliName = "dapr"
	}

	out, err := exec.Command(cliName, "--version").Output()
	if err != nil {
		return "n/a\n"
	}

	regex := regexp.MustCompile(`[0-9]+.[0-9]+.[0-9]+`)
	out = regex.Find(out)

	return string(out)

}
