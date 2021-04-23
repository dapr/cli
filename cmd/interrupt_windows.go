package cmd

import (
	"os"
	"os/exec"

	"golang.org/x/sys/windows"
)

// interruptProcess sends a CTRL-BREAK to the process.
func interruptProcess(proc *os.Process) error {
	return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(proc.Pid))
}

func suppressCtrlC(err error) error {
	if exitErr, ok := err.(*exec.ExitError); ok {
		// windows.STATUS_CONTROL_C_EXIT means
		// The application terminated as a result of a CTRL+C.
		// http://errorco.de/win32/ntstatus-h/status_control_c_exit/0xc000013a/
		if exitErr.ExitCode() == int(windows.STATUS_CONTROL_C_EXIT) {
			return nil
		}
	}

	return err
}
