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

package runconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	commonResourcesParentDir = filepath.Join(".", "app")
	commonResourcesDir       = filepath.Join(".", "app", "resources")
	app1Dir                  = filepath.Join(".", "webapp")
	app1ResourcesDir         = filepath.Join(".", "webapp", "resources")
	app1ConfigFile           = filepath.Join(".", "webapp", "config.yaml")
	app2Dir                  = filepath.Join(".", "backend")
	configFilePath           = filepath.Join("..", "testdata", "test_run_config.yaml")
)

func TestRunConfigParser(t *testing.T) {
	appsRunConfig := AppsRunConfig{}
	keyMappings, err := appsRunConfig.ParseAppsConfig(configFilePath)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(keyMappings))

	assert.Equal(t, 1, appsRunConfig.Version)
	assert.NotEmpty(t, appsRunConfig.Common.ResourcesPath)
	assert.NotEmpty(t, appsRunConfig.Common.Env)

	firstAppConfig := appsRunConfig.Apps[0]
	secondAppConfig := appsRunConfig.Apps[1]
	assert.Equal(t, "", firstAppConfig.AppID)
	assert.Equal(t, "GRPC", secondAppConfig.AppProtocol)
	assert.Equal(t, 8080, firstAppConfig.AppPort)
	assert.Equal(t, "", firstAppConfig.UnixDomainSocket)
}

func TestValidationsInRunConfig(t *testing.T) {
	// tear down the created files.
	tearDownCreatedFiles(t)

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

func TestGetApps(t *testing.T) {
	keymapping := []map[string]string{}
	keymapping = append(
		keymapping,
		map[string]string{"AppHealthTimeout": "int", "app_dir": "string", "app_port": "int", "command": "[]interface {}", "config_file": "string", "resources_dir": "string"},
		map[string]string{"app_dir": "string", "app_id": "string", "app_port": "int", "app_protocol": "string", "command": "[]interface {}", "env": "[]interface {}", "unix_domain_socket": "string"},
	)

	tearDownCreatedFiles(t)

	config := AppsRunConfig{}
	config.ParseAppsConfig(configFilePath)

	// create the files/directories provided in the config file.
	createProvidedFiles(t, config)

	apps := config.GetApps(keymapping)

	assert.Equal(t, 2, len(apps))
	assert.Equal(t, "webapp", apps[0].AppID)
	assert.Equal(t, "backend", apps[1].AppID)
	assert.Equal(t, "HTTP", apps[0].AppProtocol)
	assert.Equal(t, "GRPC", apps[1].AppProtocol)
	assert.Equal(t, 8080, apps[0].AppPort)
	assert.Equal(t, 1, apps[0].AppHealthTimeout)
	assert.Equal(t, 10, apps[1].AppHealthTimeout)
	assert.Equal(t, "", apps[0].UnixDomainSocket)
	assert.Equal(t, "/tmp/test-socket", apps[1].UnixDomainSocket)

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
	err := os.RemoveAll(commonResourcesParentDir)
	assert.Nil(t, err)
	err = os.RemoveAll(app1Dir)
	assert.Nil(t, err)
	err = os.RemoveAll(app2Dir)
	assert.Nil(t, err)
}
