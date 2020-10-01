// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	helm "helm.sh/helm/v3/pkg/action"
)

// Uninstall removes Dapr from a Kubernetes cluster.
func Uninstall(namespace string) error {
	config, err := helmConfig(namespace)
	if err != nil {
		return err
	}

	uninstallClient := helm.NewUninstall(config)
	_, err = uninstallClient.Run(daprReleaseName)
	return err
}
