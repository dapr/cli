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

// Uninstall removes Dapr from a Kubernetes cluster. releaseName is the name of
// the Helm release of the Dapr control plane; when empty, "dapr" is used.
func Uninstall(namespace string, releaseName string, uninstallAll bool, uninstallDev bool, timeout uint) error {
	if releaseName == "" {
		releaseName = daprReleaseName
	}

	config, err := helmConfig(namespace)
	if err != nil {
		return err
	}

	exists, err := confirmExist(config, releaseName)
	if err != nil {
		return err
	}

	if !exists {
		print.WarningStatusEvent(os.Stderr, "WARNING: %s release does not exist", releaseName)
		return nil
	}

	uninstallClient := helm.NewUninstall(config)
	uninstallClient.Timeout = time.Duration(timeout) * time.Second //nolint:gosec

	if uninstallDev {
		// uninstall dapr-dev-zipkin and dapr-dev-redis as best effort.
		uninstallThirdParty()
	}

	_, err = uninstallClient.Run(releaseName)
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

func uninstallThirdParty() {
	print.InfoStatusEvent(os.Stdout, "Removing dapr-dev-redis and dapr-dev-zipkin from the cluster...")
	// Uninstall dapr-dev-redis and dapr-dev-zipkin from k8s as best effort.
	config, _ := helmConfig(thirdPartyDevNamespace)

	uninstallClient := helm.NewUninstall(config)

	uninstallClient.Run(redisReleaseName)
	uninstallClient.Run(zipkinReleaseName)
}
