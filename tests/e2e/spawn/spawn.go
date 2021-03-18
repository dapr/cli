// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package spawn

import (
	"os/exec"
)

// Command runs a command with its arguments and returns the stdout or stderr or the error.
func Command(command string, arguments ...string) (string, error) {
	cmd := exec.Command(command, arguments...)

	outBytes, err := cmd.CombinedOutput()
	if err != nil && outBytes == nil {
		return "", err
	}

	return string(outBytes), err
}
