//go:build windows

package standalone

import "os/exec"

// setProcessGroup is a no-op on Windows because the syscall.SysProcAttr
// fields used on Unix (such as Setpgid) are not available.
func setProcessGroup(cmd *exec.Cmd) {
	// no-op on Windows
	// TODO: In future we should check if Windows has the same Async Python issues and address them if so.
	_ = cmd
}
