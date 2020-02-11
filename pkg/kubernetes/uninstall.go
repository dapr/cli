// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"errors"

	"github.com/dapr/cli/utils"
)

// Uninstall removes Dapr
func Uninstall() error {
	_, err := utils.RunCmdAndWait("kubectl", "delete", "-f", daprManifestPath)
	if err != nil {
		return errors.New("is Dapr running? uninstall does not remove Dapr when installed via Helm")
	}
	return nil
}
