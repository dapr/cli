// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"errors"

	"github.com/dapr/cli/utils"
)

// Uninstall the Dapr
func Uninstall() error {
	err := utils.RunCmdAndWait("kubectl", "delete", "-f", daprManifestPath)
	if err != nil {
		return errors.New("Is Dapr running? Please note uninstall does not remove Dapr when installed via Helm")
	}
	return nil
}
