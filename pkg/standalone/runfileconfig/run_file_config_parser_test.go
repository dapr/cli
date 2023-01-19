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

package runfileconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dapr/cli/pkg/standalone"

	"github.com/stretchr/testify/assert"
)

var (
	validRunFilePath    = filepath.Join("..", "testdata", "runfileconfig", "test_run_config.yaml")
	invalidRunFilePath1 = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_invalid_path.yaml")
	invalidRunFilePath2 = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_empty_app_dir.yaml")
)

func TestRunConfigFile(t *testing.T) {
	t.Run("test parse valid run template", func(t *testing.T) {
		appsRunConfig := RunFileConfig{}
		err := appsRunConfig.parseAppsConfig(validRunFilePath)

		assert.NoError(t, err)
		assert.Equal(t, 2, len(appsRunConfig.Apps))

		assert.Equal(t, 1, appsRunConfig.Version)
		assert.NotEmpty(t, appsRunConfig.Common.ResourcesPath)
		assert.NotEmpty(t, appsRunConfig.Common.Env)

		firstAppConfig := appsRunConfig.Apps[0]
		secondAppConfig := appsRunConfig.Apps[1]
		assert.Equal(t, "", firstAppConfig.AppID)
		assert.Equal(t, "GRPC", secondAppConfig.AppProtocol)
		assert.Equal(t, 8080, firstAppConfig.AppPort)
		assert.Equal(t, "", firstAppConfig.UnixDomainSocket)
	})

	t.Run("test GetApps", func(t *testing.T) {
		config := RunFileConfig{}

		apps, err := config.GetApps(validRunFilePath)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(apps))
		assert.Equal(t, "webapp", apps[0].AppID)
		assert.Equal(t, "backend", apps[1].AppID)
		assert.Equal(t, "HTTP", apps[0].AppProtocol)
		assert.Equal(t, "GRPC", apps[1].AppProtocol)
		assert.Equal(t, 8080, apps[0].AppPort)
		assert.Equal(t, 3000, apps[1].AppPort)
		assert.Equal(t, 1, apps[0].AppHealthTimeout)
		assert.Equal(t, 10, apps[1].AppHealthTimeout)
		assert.Equal(t, "", apps[0].UnixDomainSocket)
		assert.Equal(t, "/tmp/test-socket", apps[1].UnixDomainSocket)

		// test resources_path and config_path after precedence order logic.
		assert.Equal(t, filepath.Join(apps[0].AppDirPath, "resources"), apps[0].ResourcesPath)
		assert.Equal(t, filepath.Join(apps[1].AppDirPath, ".dapr", "resources"), apps[1].ResourcesPath)

		assert.Equal(t, filepath.Join(apps[0].AppDirPath, "config.yaml"), apps[0].ConfigFile)
		assert.Equal(t, filepath.Join(apps[1].AppDirPath, ".dapr", "config.yaml"), apps[1].ConfigFile)

		// temporarily set apps[0].ResourcesPath to empty string to test it is getting picked from common section.
		apps[0].ResourcesPath = ""
		config.resolveResourcesAndConfigFilePaths()
		assert.Equal(t, config.Common.ResourcesPath, apps[0].ResourcesPath)
		assert.Equal(t, filepath.Join(apps[1].AppDirPath, ".dapr", "resources"), apps[1].ResourcesPath)

		// test merged envs from common and app sections.
		assert.Equal(t, 2, len(apps[0].Env))
		assert.Equal(t, 2, len(apps[1].Env))
		assert.Equal(t, "DEBUG", apps[0].Env[0].Name)
		assert.Equal(t, "false", apps[0].Env[0].Value)
		assert.Equal(t, "DEBUG", apps[1].Env[0].Name)
		assert.Equal(t, "true", apps[1].Env[0].Value)
	})

	t.Run("test precedence logic for resources-path and dapr config file", func(t *testing.T) {
		config := RunFileConfig{}

		err := config.parseAppsConfig(validRunFilePath)
		assert.NoError(t, err)
		err = config.validateRunConfig(validRunFilePath)
		assert.NoError(t, err)

		daprDirPath, err := standalone.GetDaprPath(config.Apps[0].DaprdInstallPath)
		assert.NoError(t, err)
		configFilePath := standalone.GetDaprConfigPath(daprDirPath)

		// temporarily set dapr's installation directory to Config.Apps[1].AppDirPath.
		tDaprDirPath, err := standalone.GetDaprPath(config.Apps[1].AppDirPath)
		assert.NoError(t, err)
		configFilePathWithCustomDaprdDir := standalone.GetDaprConfigPath(tDaprDirPath)

		testcases := []struct {
			name                      string
			setResourcesAndConfigPath bool
			setDaprdInstallPath       bool
			expectedResourcesPath     string
			expectedConfigFilePath    string
			appIndex                  int
		}{
			{
				name:                      "resources_path and config_path is set in app section",
				setResourcesAndConfigPath: false,
				setDaprdInstallPath:       false,
				expectedResourcesPath:     filepath.Join(config.Apps[0].AppDirPath, "resources"),
				expectedConfigFilePath:    filepath.Join(config.Apps[0].AppDirPath, "config.yaml"),
				appIndex:                  0,
			},
			{
				name:                      "resources_path and config_path present in .dapr directory under appDirPath",
				setResourcesAndConfigPath: false,
				setDaprdInstallPath:       false,
				expectedResourcesPath:     filepath.Join(config.Apps[1].AppDirPath, ".dapr", "resources"),
				expectedConfigFilePath:    filepath.Join(config.Apps[1].AppDirPath, ".dapr", "config.yaml"),
				appIndex:                  1,
			},
			{
				name:                      "temporarily set ResourcesPath and configPath to empty string in app's section",
				setResourcesAndConfigPath: true,
				setDaprdInstallPath:       false,
				expectedResourcesPath:     config.Common.ResourcesPath, // from common section.
				expectedConfigFilePath:    configFilePath,              // from dapr's installation directory.
				appIndex:                  0,
			},
			{
				name:                      "custom dapr install path with empty ResourcesPath and configPath in app's section",
				setResourcesAndConfigPath: true,
				setDaprdInstallPath:       true,
				expectedResourcesPath:     config.Common.ResourcesPath,      // from common section.
				expectedConfigFilePath:    configFilePathWithCustomDaprdDir, // from dapr's custom installation directory.
				appIndex:                  0,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.setResourcesAndConfigPath {
					config.Apps[tc.appIndex].ResourcesPath = ""
					config.Apps[tc.appIndex].ConfigFile = ""
				}
				if tc.setDaprdInstallPath {
					config.Apps[tc.appIndex].DaprdInstallPath = config.Apps[1].AppDirPath // set to apps[1].AppDirPath teporarily.
				}
				// test precedence logic for resources_path and config_path.
				config.resolveResourcesAndConfigFilePaths()
				assert.Equal(t, tc.expectedResourcesPath, config.Apps[tc.appIndex].ResourcesPath)
				assert.Equal(t, tc.expectedConfigFilePath, config.Apps[tc.appIndex].ConfigFile)
			})
		}
	})

	t.Run("test validate run config", func(t *testing.T) {
		testcases := []struct {
			name        string
			input       string
			expectedErr bool
		}{
			{
				name:        "valid run config",
				input:       validRunFilePath,
				expectedErr: false,
			},
			{
				name:        "invalid run config - empty app dir path",
				input:       invalidRunFilePath2,
				expectedErr: true,
			},
			{
				name:        "invalid run config - invalid path",
				input:       invalidRunFilePath1,
				expectedErr: true,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				config := RunFileConfig{}
				config.parseAppsConfig(tc.input)
				actualErr := config.validateRunConfig(tc.input)
				assert.Equal(t, tc.expectedErr, actualErr != nil)
			})
		}
	})
}

func TestGetBasePathFromAbsPath(t *testing.T) {
	testcases := []struct {
		name          string
		input         string
		expectedErr   bool
		expectedAppID string
	}{
		{
			name:          "valid absolute path",
			input:         filepath.Join(os.TempDir(), "test"),
			expectedErr:   false,
			expectedAppID: "test",
		},
		{
			name:          "invalid absolute path",
			input:         filepath.Join("..", "test"),
			expectedErr:   true,
			expectedAppID: "",
		},
		{
			name:          "invalid absolute path",
			input:         filepath.Join(".", "test"),
			expectedErr:   true,
			expectedAppID: "",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			runFileConfig := RunFileConfig{}
			appID, actualErr := runFileConfig.getBasePathFromAbsPath(tc.input)
			assert.Equal(t, tc.expectedErr, actualErr != nil)
			assert.Equal(t, tc.expectedAppID, appID)
		})
	}
}
