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

package standalone

import "os/exec"

// setProcessGroup is a no-op on Windows because the syscall.SysProcAttr
// fields used on Unix (such as Setpgid) are not available.
func setProcessGroup(cmd *exec.Cmd) {
	// no-op on Windows
	// TODO: In future we should check if Windows has the same Async Python issues and address them if so.
	_ = cmd
}
