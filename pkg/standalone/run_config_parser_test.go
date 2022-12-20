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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	commonResourcesDir = "./app/resources"
	app1Dir            = "./webapp/"
	app1ResourcesDir   = "./webapp/resources"
	app1ConfigFile     = "./webapp/config.yaml"
	app2Dir            = "./backend/"
)

func TestRunConfigParser(t *testing.T) {
	configFilePath := "./testdata/test_run_config.yaml"
	appsRunConfig := AppsRunConfig{}
	appsRunConfig.ParseAppsConfig(configFilePath)

	assert.Equal(t, 1, appsRunConfig.Version)
	assert.NotEmpty(t, appsRunConfig.Common.ResourcesPath)
	assert.NotEmpty(t, appsRunConfig.Common.Env)

	firstAppConfig := appsRunConfig.Apps[0]
	assert.Equal(t, "", firstAppConfig.AppID)
	assert.Equal(t, "HTTP", firstAppConfig.AppProtocol)
	assert.Equal(t, 8080, firstAppConfig.AppPort)
	assert.Equal(t, "", firstAppConfig.UnixDomainSocket)
}

func TestValidationsInRunConfig(t *testing.T) {
	// tear down the created files.
	tearDownCreatedFiles(t)

	configFilePath := "./testdata/test_run_config.yaml"
	config := AppsRunConfig{}
	config.ParseAppsConfig(configFilePath)

	// check mangatory fields are not empty.
	for _, app := range config.Apps {
		assert.NotEmpty(t, app.AppDir)
	}

	// provided files/directories does not exist.
	err := config.ValidateRunConfig()
	assert.NotNil(t, err)

	// create the files/directories provided in the config file.
	createProvidedFiles(t, config)

	// positive case- all files and directories exist.
	err = config.ValidateRunConfig()
	assert.Nil(t, err)

	// negative case- app-dir field is empty.
	temp := config.Apps[0].AppDir
	config.Apps[0].AppDir = ""
	err = config.ValidateRunConfig()
	assert.NotNil(t, err)
	config.Apps[0].AppDir = temp

	// tear down the created files.
	tearDownCreatedFiles(t)
}

func createProvidedFiles(t *testing.T, config AppsRunConfig) {
	err := os.MkdirAll(commonResourcesDir, os.ModePerm)
	assert.Nil(t, err)
	err = os.MkdirAll(app1ResourcesDir, os.ModePerm)
	assert.Nil(t, err)
	err = os.MkdirAll(app1Dir, os.ModePerm)
	assert.Nil(t, err)
	err = os.MkdirAll(app2Dir, os.ModePerm)
	assert.Nil(t, err)
	_, err = os.Create(app1ConfigFile)
	assert.Nil(t, err)
}

func tearDownCreatedFiles(t *testing.T) {
	err := os.RemoveAll(commonResourcesDir)
	assert.Nil(t, err)
	err = os.RemoveAll(app1Dir)
	assert.Nil(t, err)
	err = os.RemoveAll(app2Dir)
	assert.Nil(t, err)
}
