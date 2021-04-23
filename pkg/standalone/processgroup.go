//go:build !windows
// +build !windows

// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"os/exec"
)

func setCmdSysProcAttr(cmd *exec.Cmd) {
	// Do nothing
}
