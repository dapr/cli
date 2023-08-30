/*
Copyright 2023 The Dapr Authors
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
	"context"
	"errors"
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"

	"github.com/dapr/cli/pkg/standalone/runfileconfig"
)

func Stop(runFilePath string, config runfileconfig.RunFileConfig) error {
	errs := []error{}
	// get k8s client.
	client, cErr := Client()
	if cErr != nil {
		return fmt.Errorf("error getting k8s client for monitoring pod deletion: %w", cErr)
	}

	namespace := corev1.NamespaceDefault
	for _, app := range config.Apps {
		deployDir := app.GetDeployDir()
		serviceFilePath := filepath.Join(deployDir, serviceFileName)
		deploymentFilePath := filepath.Join(deployDir, deploymentFileName)
		if app.CreateService {
			errs = append(errs, deleteYamlK8s(serviceFilePath))
		}
		errs = append(errs, deleteYamlK8s(deploymentFilePath))
		ctx, cancel := context.WithTimeout(context.Background(), podCreationDeletionTimeout)

		// Ignoring errors here as it will anyway be printed in the other dapr cli process.
		waitPodDeleted(ctx, client, namespace, app.AppID)
		cancel()
	}
	return errors.Join(errs...)
}