// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package version

import (
	"os/exec"
	"runtime"
)

// GetRuntimeVersion returns the version for the local Dapr runtime
func GetRuntimeVersion() string {
	runtimeName := ""
	if runtime.GOOS == "windows" {
		runtimeName = "daprd.exe"
	} else {
		runtimeName = "daprd"
	}

	out, err := exec.Command(runtimeName, "--version").Output()
	if err != nil {
		return "n/a"
	}
	return string(out)
}
