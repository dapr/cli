//go:build !windows
// +build !windows

package cmd

import (
	"os"
	"os/exec"
	"syscall"
)

// interruptProcess sends an Interrupt signal to the process.
func interruptProcess(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}

func suppressCtrlC(err error) error {
	if exitErr, ok := err.(*exec.ExitError); ok {
		// The SIGINT signal is sent when the user at the
		// controlling terminal presses the interrupt character,
		// which by default is ^C (Control-C)
		// https://golang.org/pkg/os/signal/#hdr-Types_of_signals
		if exitErr.ExitCode() == int(syscall.SIGINT) {
			return nil
		}
	}

	return err
}
