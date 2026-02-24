//go:build !windows

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

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/dapr/cli/pkg/print"
	runExec "github.com/dapr/cli/pkg/runexec"
)

// setDaprProcessGroupForRun sets the process group on the daprd command so the
// sidecar can be managed independently (e.g. when the app is started via exec).
func setDaprProcessGroupForRun(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// startAppProcessInBackground starts the app process using ForkExec.
// This prevents the child from seeing a fork, avoiding Python async/threading issues,
// and sets output.AppCMD.Process.
// It then runs a goroutine that waits and signals sigCh.
func startAppProcessInBackground(output *runExec.RunOutput, binary string, args []string, env []string, sigCh chan os.Signal) error {
	if output.AppCMD == nil || output.AppCMD.Process != nil {
		return fmt.Errorf("app command is nil")
	}

	procAttr := &syscall.ProcAttr{
		Env: env,
		// stdin, stdout, and stderr inherit directly from the parent
		// This prevents Python from detecting pipes because if the app is Python then it will detect the pipes and think
		// it's a fork and will cause random hangs due to async python in durabletask-python.
		Files: []uintptr{0, 1, 2},
		Sys: &syscall.SysProcAttr{
			Setpgid: true,
		},
	}

	// Use ForkExec to fork a child, then exec python in the child.
	// NOTE: This is needed bc forking a python app with async python running (i.e., everything in durabletask-python) will cause random hangs, no matter the python version.
	// Doing this this way makes python not see the fork, starts via exec, so it doesn't cause random hangs due to when forking async python apps where locks and such get corrupted in forking.
	argv := append([]string{binary}, args[1:]...)
	pid, err := syscall.ForkExec(binary, argv, procAttr)
	if err != nil {
		return fmt.Errorf("failed to fork/exec app: %w", err)
	}
	output.AppCMD.Process = &os.Process{Pid: pid}

	go func() {
		var waitStatus syscall.WaitStatus
		_, err := syscall.Wait4(pid, &waitStatus, 0, nil)
		if err != nil {
			output.AppErr = err
			print.FailureStatusEvent(os.Stderr, "The App process exited with error: %s", err.Error())
		} else if waitStatus.Signaled() {
			output.AppErr = fmt.Errorf("app terminated by signal: %s", waitStatus.Signal())
			print.FailureStatusEvent(os.Stderr, "The App process was terminated by signal: %s", waitStatus.Signal())
		} else if waitStatus.Exited() && waitStatus.ExitStatus() != 0 {
			output.AppErr = fmt.Errorf("app exited with status %d", waitStatus.ExitStatus())
			print.FailureStatusEvent(os.Stderr, "The App process exited with error code: %d", waitStatus.ExitStatus())
		} else {
			print.SuccessStatusEvent(os.Stdout, "Exited App successfully")
		}
		sigCh <- os.Interrupt
	}()
	return nil
}
