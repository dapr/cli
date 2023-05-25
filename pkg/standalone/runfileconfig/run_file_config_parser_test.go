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
	"strings"
	"testing"

	"github.com/dapr/cli/pkg/standalone"

	"github.com/stretchr/testify/assert"
)

var (
	validRunFilePath                = filepath.Join("..", "testdata", "runfileconfig", "test_run_config.yaml")
	invalidRunFilePath1             = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_invalid_path.yaml")
	invalidRunFilePath2             = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_empty_app_dir.yaml")
	runFileForPrecedenceRule        = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_precedence_rule.yaml")
	runFileForPrecedenceRuleDaprDir = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_precedence_rule_dapr_dir.yaml")
	runFileForLogDestination        = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_log_destination.yaml")
	runFileForMultiResourcePaths    = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_multiple_resources_paths.yaml")
)

func TestRunConfigFile(t *testing.T) {
	t.Parallel()
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

		// test resourcesPath and configPath after precedence order logic.
		assert.Equal(t, filepath.Join(apps[0].AppDirPath, "resources"), apps[0].ResourcesPaths[0])
		assert.Equal(t, filepath.Join(apps[1].AppDirPath, ".dapr", "resources"), apps[1].ResourcesPaths[0])

		assert.Equal(t, filepath.Join(apps[0].AppDirPath, "config.yaml"), apps[0].ConfigFile)
		assert.Equal(t, filepath.Join(apps[1].AppDirPath, ".dapr", "config.yaml"), apps[1].ConfigFile)

		// temporarily set apps[0].ResourcesPath to empty string to test it is getting picked from common section.
		apps[0].ResourcesPaths = []string{}
		config.resolveResourcesAndConfigFilePaths()
		assert.Equal(t, config.Common.ResourcesPaths[0], apps[0].ResourcesPaths[0])
		assert.Equal(t, filepath.Join(apps[1].AppDirPath, ".dapr", "resources"), apps[1].ResourcesPaths[0])

		// test merged envs from common and app sections.
		assert.Equal(t, 2, len(apps[0].Env))
		assert.Equal(t, 2, len(apps[1].Env))
		assert.Contains(t, apps[0].Env, "DEBUG")
		assert.Contains(t, apps[0].Env, "tty")
		assert.Contains(t, apps[1].Env, "DEBUG")
		assert.Contains(t, apps[1].Env, "tty")
		assert.Equal(t, "false", apps[0].Env["DEBUG"])
		assert.Equal(t, "sts", apps[0].Env["tty"])
		assert.Equal(t, "true", apps[1].Env["DEBUG"])
		assert.Equal(t, "sts", apps[1].Env["tty"])
	})

	t.Run("test precedence logic for resources-path and dapr config file", func(t *testing.T) {
		config := RunFileConfig{}

		err := config.parseAppsConfig(runFileForPrecedenceRule)
		assert.NoError(t, err)
		err = config.validateRunConfig(runFileForPrecedenceRule)
		assert.NoError(t, err)

		// test precedence logic for resourcesPath and configPath.
		err = config.resolveResourcesAndConfigFilePaths()
		assert.NoError(t, err)

		testcases := []struct {
			name                   string
			expectedResourcesPath  string
			expectedConfigFilePath string
			appIndex               int
		}{
			{
				name:                   "resourcesPaths and configPath are set in app section",
				expectedResourcesPath:  filepath.Join(config.Apps[0].AppDirPath, "resources"),
				expectedConfigFilePath: filepath.Join(config.Apps[0].AppDirPath, "config.yaml"),
				appIndex:               0,
			},
			{
				name:                   "resourcesPaths and configPath present in .dapr directory under appDirPath",
				expectedResourcesPath:  filepath.Join(config.Apps[1].AppDirPath, ".dapr", "resources"),
				expectedConfigFilePath: filepath.Join(config.Apps[1].AppDirPath, ".dapr", "config.yaml"),
				appIndex:               1,
			},
			{
				name:                   "resourcesPaths and configPath are resolved from common's section",
				expectedResourcesPath:  config.Common.ResourcesPaths[0], // from common section.
				expectedConfigFilePath: config.Common.ConfigFile,        // from common section.
				appIndex:               2,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.expectedResourcesPath, config.Apps[tc.appIndex].ResourcesPaths[0])
				assert.Equal(t, tc.expectedConfigFilePath, config.Apps[tc.appIndex].ConfigFile)
			})
		}
	})

	t.Run("test precedence logic with daprInstallDir for resources-path and dapr config file", func(t *testing.T) {
		config := RunFileConfig{}

		err := config.parseAppsConfig(runFileForPrecedenceRuleDaprDir)
		assert.NoError(t, err)
		err = config.validateRunConfig(runFileForPrecedenceRuleDaprDir)
		assert.NoError(t, err)

		app1Data := getResourcesAndConfigFilePaths(t, config.Apps[0].DaprdInstallPath)
		app1ResourcesPath := app1Data[0]
		app1ConfigFilePath := app1Data[1]

		app2Data := getResourcesAndConfigFilePaths(t, config.Apps[1].DaprdInstallPath)
		app2ResourcesPath := app2Data[0]
		app2ConfigFilePath := app2Data[1]

		// test precedence logic for resourcesPath and configPath.
		err = config.resolveResourcesAndConfigFilePaths()
		assert.NoError(t, err)

		testcases := []struct {
			name                   string
			expectedResourcesPath  string
			expectedConfigFilePath string
			appIndex               int
		}{
			{
				name:                   "resourcesPaths and configPath are resolved from dapr's default installation path.",
				expectedResourcesPath:  app1ResourcesPath,
				expectedConfigFilePath: app1ConfigFilePath,
				appIndex:               0,
			},
			{
				name:                   "resourcesPaths and configPath are resolved from dapr's custom installation path.",
				expectedResourcesPath:  app2ResourcesPath,
				expectedConfigFilePath: app2ConfigFilePath,
				appIndex:               1,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.expectedResourcesPath, config.Apps[tc.appIndex].ResourcesPaths[0])
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
				name:        "invalid run config - invalid app dir path",
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

	t.Run("test log destination for daprd and apps", func(t *testing.T) {
		config := RunFileConfig{}
		apps, err := config.GetApps(runFileForLogDestination)
		assert.NoError(t, err)
		assert.Equal(t, 6, len(apps))

		assert.Equal(t, "file", apps[0].DaprdLogDestination.String())
		assert.Equal(t, "fileAndConsole", apps[0].AppLogDestination.String())

		assert.Equal(t, "fileAndConsole", apps[1].DaprdLogDestination.String())
		assert.Equal(t, "fileAndConsole", apps[1].AppLogDestination.String())

		assert.Equal(t, "file", apps[2].DaprdLogDestination.String())
		assert.Equal(t, "file", apps[2].AppLogDestination.String())

		assert.Equal(t, "console", apps[3].DaprdLogDestination.String())
		assert.Equal(t, "console", apps[3].AppLogDestination.String())

		assert.Equal(t, "console", apps[4].DaprdLogDestination.String())
		assert.Equal(t, "file", apps[4].AppLogDestination.String())

		assert.Equal(t, "file", apps[5].DaprdLogDestination.String())
		assert.Equal(t, "console", apps[5].AppLogDestination.String())
	})
}

func TestMultiResourcePathsResolution(t *testing.T) {
	config := RunFileConfig{}

	err := config.parseAppsConfig(runFileForMultiResourcePaths)
	assert.NoError(t, err)
	err = config.validateRunConfig(runFileForMultiResourcePaths)
	assert.NoError(t, err)

	// test precedence logic for multiple resources paths.
	err = config.resolveResourcesAndConfigFilePaths()
	assert.NoError(t, err)

	testcases := []struct {
		name                           string
		expectedNoOfResources          int
		expectedResourcesPathsContains string
		appIndex                       int
	}{
		{
			name:                           "resourcesPaths should have 2 paths",
			expectedNoOfResources:          2,
			expectedResourcesPathsContains: filepath.Join(config.Apps[0].AppDirPath, "resources"),
			appIndex:                       0,
		},
		{
			name:                           "resourcesPaths should have 2 paths",
			expectedNoOfResources:          2,
			expectedResourcesPathsContains: filepath.Join("backend", ".dapr", "resources"),
			appIndex:                       0,
		},
		{
			name:                           "resourcesPaths should have 2 path from common section",
			expectedNoOfResources:          2,
			expectedResourcesPathsContains: filepath.Join("app", "resources"),
			appIndex:                       1,
		},
		{
			name:                           "resourcesPaths should have 1 path from .dapr's folder",
			expectedNoOfResources:          1,
			expectedResourcesPathsContains: filepath.Join("backend", ".dapr", "resources"),
			appIndex:                       2,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedNoOfResources, len(config.Apps[tc.appIndex].ResourcesPaths))
			var rsrcFound bool
			for _, resourcePath := range config.Apps[tc.appIndex].ResourcesPaths {
				if rsrcFound = strings.Contains(resourcePath, tc.expectedResourcesPathsContains); rsrcFound {
					break
				}
			}
			assert.True(t, rsrcFound)
		})
	}
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

// getResoucresAndConfigFilePaths returns a list containing resources and config file paths in order.
func getResourcesAndConfigFilePaths(t *testing.T, daprInstallPath string) []string {
	t.Helper()
	result := make([]string, 2)
	daprDirPath, err := standalone.GetDaprRuntimePath(daprInstallPath)
	assert.NoError(t, err)
	result[0] = standalone.GetDaprComponentsPath(daprDirPath)
	result[1] = standalone.GetDaprConfigPath(daprDirPath)
	return result
}
