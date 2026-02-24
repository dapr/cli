//go:build windows

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

	"github.com/dapr/cli/pkg/print"
	runExec "github.com/dapr/cli/pkg/runexec"
)

// setDaprProcessGroupForRun is a no-op on Windows (SysProcAttr.Setpgid does not exist).
func setDaprProcessGroupForRun(cmd *exec.Cmd) {
	// no-op on Windows
	_ = cmd
}

// startAppProcessInBackground starts the app process using exec.Command,
// sets output.AppCMD to the new command, and runs a goroutine that waits and signals sigCh.
func startAppProcessInBackground(output *runExec.RunOutput, binary string, args []string, env []string, sigCh chan os.Signal) error {
	cmdArgs := args[1:]
	if output.AppCMD == nil {
		output.AppCMD = exec.Command(binary, cmdArgs...)
	} else {
		output.AppCMD.Path = binary
		output.AppCMD.Args = append([]string{binary}, cmdArgs...)
	}
	output.AppCMD.Env = env
	output.AppCMD.Stdin = os.Stdin
	output.AppCMD.Stdout = os.Stdout
	output.AppCMD.Stderr = os.Stderr

	if err := output.AppCMD.Start(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	go func() {
		waitErr := output.AppCMD.Wait()
		if waitErr != nil {
			output.AppErr = waitErr
			print.FailureStatusEvent(os.Stderr, "The App process exited with error: %s", waitErr.Error())
		} else if output.AppCMD.ProcessState != nil && !output.AppCMD.ProcessState.Success() {
			output.AppErr = fmt.Errorf("app exited with status %d", output.AppCMD.ProcessState.ExitCode())
			print.FailureStatusEvent(os.Stderr, "The App process exited with error code: %d", output.AppCMD.ProcessState.ExitCode())
		} else {
			print.SuccessStatusEvent(os.Stdout, "Exited App successfully")
		}
		sigCh <- os.Interrupt
	}()
	return nil
}
