/*
Copyright 2022 The Dapr Authors
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

package standalone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunConfigParser(t *testing.T) {
	configFilePath := "./testdata/test_run_config.yaml"
	appsRunConfig := AppsRunConfig{}
	appsRunConfig.ParseAppsConfig(configFilePath)

	assert.Equal(t, 1, appsRunConfig.Version)
	assert.NotEmpty(t, appsRunConfig.Common.ResourcesPath)
	assert.NotEmpty(t, appsRunConfig.Common.Env)

	firstAppConfig := appsRunConfig.Apps[0]
	assert.Equal(t, "webapp", firstAppConfig.AppID)
	assert.Equal(t, "HTTP", firstAppConfig.AppProtocol)
	assert.Equal(t, 8080, firstAppConfig.AppPort)
	assert.Equal(t, "", firstAppConfig.UnixDomainSocket)
}

func TestMandatoryFieldsInRunConfig(t *testing.T) {
	configFilePath := "./testdata/test_run_config.yaml"
	config := AppsRunConfig{}
	config.ParseAppsConfig(configFilePath)

	assert.Equal(t, 1, config.Version)
	assert.NotEmpty(t, config.Common.ResourcesPath)
	assert.NotEmpty(t, config.Common.Env)

	for _, app := range config.Apps {
		assert.NotEmpty(t, app.AppDir)
		assert.NotEmpty(t, app.AppID)
	}

	err := config.ValidateRunConfig()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}
