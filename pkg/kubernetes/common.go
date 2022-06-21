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
	"errors"
	"strings"

	"github.com/dapr/cli/utils"

	helm "helm.sh/helm/v3/pkg/action"
)

const (
	dockerContainerRegistryName = "dockerhub"
	githubContainerRegistryName = "ghcr"
	ghcrURI                     = "ghcr.io/dapr"
)

func GetDaprResourcesStatus() ([]StatusOutput, error) {
	sc, err := NewStatusClient()
	if err != nil {
		return nil, err
	}

	status, err := sc.Status()
	if err != nil {
		return nil, err
	}

	if len(status) == 0 {
		return nil, errors.New("dapr is not installed in your cluster")
	}
	return status, nil
}

func GetDaprHelmChartName(helmConf *helm.Configuration) (string, error) {
	listClient := helm.NewList(helmConf)
	releases, err := listClient.Run()
	if err != nil {
		return "", err
	}
	var chart string
	for _, r := range releases {
		if r.Chart != nil && strings.Contains(r.Chart.Name(), "dapr") {
			chart = r.Name
			break
		}
	}
	return chart, nil
}

func GetDaprVersion(status []StatusOutput) string {
	var daprVersion string
	for _, s := range status {
		if s.Name == operatorName {
			daprVersion = s.Version
		}
	}
	return daprVersion
}

func GetDaprNamespace() (string, error) {
	status, err := GetDaprResourcesStatus()
	if err != nil {
		return "", err
	}
	return status[0].Namespace, nil
}

func GetImageRegistry() (string, error) {
	defaultImageRegistry, err := utils.GetDefaultRegistry(githubContainerRegistryName, dockerContainerRegistryName)
	if err != nil {
		return "", err
	}
	if defaultImageRegistry == githubContainerRegistryName {
		return ghcrURI, nil
	}
	return "", nil
}
