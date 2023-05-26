/*
Copyright 2021 The Dapr Authors
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

package kubernetes

import (
	"os"
	"time"

	helm "helm.sh/helm/v3/pkg/action"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

// Uninstall removes Dapr from a Kubernetes cluster.
func Uninstall(namespace string, uninstallAll bool, timeout uint) error {
	config, err := helmConfig(namespace)
	if err != nil {
		return err
	}

	exists, err := confirmExist(config, daprReleaseName)
	if err != nil {
		return err
	}

	if !exists {
		print.WarningStatusEvent(os.Stderr, "WARNING: %s release does not exist", daprReleaseName)
		return nil
	}

	uninstallClient := helm.NewUninstall(config)
	uninstallClient.Timeout = time.Duration(timeout) * time.Second

	// Uninstall Dashboard as a best effort.
	// Chart versions < 1.11 for Dapr will delete dashboard as part of the main chart.
	// Deleting Dashboard here is for versions >= 1.11.
	uninstallClient.Run(dashboardReleaseName)

	_, err = uninstallClient.Run(daprReleaseName)

	if err != nil {
		return err
	}

	if uninstallAll {
		for _, crd := range crdsFullResources {
			_, err := utils.RunCmdAndWait("kubectl", "delete", "crd", crd)
			if err != nil {
				print.WarningStatusEvent(os.Stdout, "Failed to remove CRD %s: %s", crd, err)
			}
		}
	}

	return nil
}
