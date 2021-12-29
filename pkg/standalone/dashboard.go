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
