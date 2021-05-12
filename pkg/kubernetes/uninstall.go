// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"time"

	"github.com/dapr/cli/utils"
	helm "helm.sh/helm/v3/pkg/action"
)

var crdsFullResource = []string{
	"configurations.dapr.io",
	"components.dapr.io",
	"subscriptions.dapr.io",
}

// Uninstall removes Dapr from a Kubernetes cluster.
func Uninstall(namespace string, timeout uint) error {
	config, err := helmConfig(namespace)
	if err != nil {
		return err
	}

	uninstallClient := helm.NewUninstall(config)
	uninstallClient.Timeout = time.Duration(timeout) * time.Second
	_, err = uninstallClient.Run(daprReleaseName)
	if err != nil {
		return err
	}

	err = removeCRDs()
	if err != nil {
		return err
	}

	return err
}

func removeCRDs() error {
	for _, crd := range crdsFullResource {
		_, err := utils.RunCmdAndWait("kubectl", "delete", "crd", crd)
		if err != nil {
			return err
		}
	}

	return nil
}
