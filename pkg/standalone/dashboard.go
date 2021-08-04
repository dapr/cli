// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

// NewDashboardCmd creates the command to run dashboard.
func NewDashboardCmd(port int) *exec.Cmd {
	// Use the default binary install location
	dashboardPath := defaultDaprBinPath()
	binaryName := "dashboard"
	if runtime.GOOS == daprWindowsOS {
		binaryName = "dashboard.exe"
	}

	// Construct command to run dashboard
	return &exec.Cmd{
		Path:   filepath.Join(dashboardPath, binaryName),
		Args:   []string{binaryName, "--port", strconv.Itoa(port)},
		Dir:    dashboardPath,
		Stdout: os.Stdout,
	}
}
