// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import "errors"

// Uninstall the Dapr
func Uninstall() error {
	err := runCmdAndWait("kubectl", "delete", "-f", daprManifestPath)
	if err != nil {
		return errors.New("Is Dapr running? Please note uninstall does not remove Dapr when installed via Helm")
	}
	return nil
}
