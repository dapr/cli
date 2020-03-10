// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"github.com/dapr/cli/utils"
)

// Uninstall removes Dapr
func Uninstall() error {
	//TODO: change how we handle init and uninstall for Kubernetes
	utils.RunCmdAndWait("kubectl", "delete", "-f", daprManifestPath)
	return nil
}
