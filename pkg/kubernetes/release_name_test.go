/*
Copyright 2026 The Dapr Authors
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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

func TestInitConfigurationReleaseName(t *testing.T) {
	assert.Equal(t, "dapr", InitConfiguration{}.releaseName())
	assert.Equal(t, "my-dapr", InitConfiguration{ReleaseName: "my-dapr"}.releaseName())
}

func TestConfirmExistCustomReleaseName(t *testing.T) {
	cfg := &helm.Configuration{
		Releases:     storage.Init(driver.NewMemory()),
		KubeClient:   &kubefake.PrintingKubeClient{Out: io.Discard},
		Capabilities: chartutil.DefaultCapabilities,
	}
	require.NoError(t, cfg.Releases.Create(&release.Release{
		Name:      "my-dapr",
		Namespace: "dapr-system",
		Version:   1,
		Info:      &release.Info{Status: release.StatusDeployed},
		Chart:     &chart.Chart{Metadata: &chart.Metadata{Name: daprChartName}},
	}))

	exists, err := confirmExist(cfg, "my-dapr")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = confirmExist(cfg, daprReleaseName)
	require.NoError(t, err)
	assert.False(t, exists)

	name, err := GetDaprHelmChartName(cfg)
	require.NoError(t, err)
	assert.Equal(t, "my-dapr", name)
}
