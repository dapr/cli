// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import(
	"fmt"

	"github.com/dapr/cli/utils"
)

func InitDashboard() error {
	
	var daprDashboardManifestPath string = "https://raw.githubusercontent.com/dapr/dashboard/master/deploy/dashboard.yaml"


	_, err := utils.RunCmdAndWait("kubectl", "apply", "-f", daprDashboardManifestPath)
	if err != nil {
		return fmt.Errorf("Failed to init Dapr dashboard")
	}

	return nil
}