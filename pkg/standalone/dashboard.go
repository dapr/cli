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
	"strconv"

	"github.com/phayes/freeport"
)

// NewDashboardCmd creates the command to run dashboard.
func NewDashboardCmd(inputInstallPath string, port int) (*exec.Cmd, error) {
	if port == 0 {
		freePort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}
		port = freePort
	}
	dashboardPath, err := lookupBinaryFilePath(inputInstallPath, "dashboard")
	if err != nil {
		return nil, err
	}

	// Construct command to run dashboard.
	return &exec.Cmd{
		Path:   dashboardPath,
		Args:   []string{filepath.Base(dashboardPath), "--port", strconv.Itoa(port)},
		Dir:    filepath.Dir(dashboardPath),
		Stdout: os.Stdout,
	}, nil
}
