//go:build !windows
// +build !windows

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

package syscall

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/dapr/cli/pkg/print"
)

func SetupShutdownNotify(sigCh chan os.Signal) {
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
}

// CreateProcessGroupID creates a process group ID for the current process.
// Reference - https://man7.org/linux/man-pages/man2/setpgid.2.html.
func CreateProcessGroupID() {
	// Below is some excerpt from the above link Setpgid() -
	// setpgid() sets the PGID of the process specified by pid to pgid.
	// If pid is zero, then the process ID of the calling process is
	// used.  If pgid is zero, then the PGID of the process specified by
	// pid is made the same as its process ID.
	if err := syscall.Setpgid(0, 0); err != nil {
		print.WarningStatusEvent(os.Stdout, "Failed to create process group id: %s", err.Error())
	}
}

// AttachJobObjectToProcess attaches the process to a job object.
func AttachJobObjectToProcess(jobName string, proc *os.Process) {
	// This is a no-op on Linux/Mac.
	// Instead, we use process group ID to kill all the processes.
}
