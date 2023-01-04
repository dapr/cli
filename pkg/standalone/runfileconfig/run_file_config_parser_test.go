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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	validRunFilePath    = filepath.Join("..", "testdata", "runfileconfig", "test_run_config.yaml")
	invalidRunFilePath1 = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_invalid_path.yaml")
	invalidRunFilePath2 = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_empty_app_dir.yaml")
)

func TestRunConfigFile(t *testing.T) {
	tearDownFn := testSetup(t)
	defer tearDownFn(t)

	t.Run("test run config parser", func(t *testing.T) {
		appsRunConfig := RunFileConfig{}
		err := appsRunConfig.parseAppsConfig(validRunFilePath)

		assert.Nil(t, err)
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

	t.Run("test run GetApps", func(t *testing.T) {
		config := RunFileConfig{}

		apps, err := config.GetApps(validRunFilePath)
		assert.Nil(t, err)
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
	t.Run("test validate config files", func(t *testing.T) {
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

// Expected directory and file structures from this method:
// 1) testdata/runfileconfig/app/resources.
// 2) testdata/runfileconfig/backend/.dapr/resources.
// 3) testdata/runfileconfig/backend/.dapr/config.yaml.
// 4) testdata/runfileconfig/webapp/resources.
// 5) testdata/runfileconfig/webapp/config.yaml.
func testSetup(t *testing.T) func(t *testing.T) {
	baseDir := filepath.Join("..", "testdata", "runfileconfig")

	// These paths are according to the values present in the testdata/runfileconfig/test_run_config.yaml file.
	commonResourcesPath := filepath.Join(baseDir, "app", "resources")

	backendAppResourcesPath := filepath.Join(baseDir, "backend", ".dapr", "resources")
	backendAppConfigPath := filepath.Join(baseDir, "backend", ".dapr", "config.yaml")

	webappAppResourcesPath := filepath.Join(baseDir, "webapp", "resources")
	webAppAppConfigPath := filepath.Join(baseDir, "webapp", "config.yaml")

	err := createDirPath(commonResourcesPath, backendAppResourcesPath, webappAppResourcesPath)
	assert.Nil(t, err)

	err = createFile(webAppAppConfigPath, backendAppConfigPath)
	assert.Nil(t, err)

	return func(t *testing.T) {
		err := removeAllDirPaths(filepath.Join(baseDir, "app"), filepath.Join(baseDir, "backend"), filepath.Join(baseDir, "webapp"))
		assert.Nil(t, err)
	}
}

// Helper function to create slice of directory paths.
func createDirPath(dirPaths ...string) error {
	for _, path := range dirPaths {
		if path != "" {
			if err := os.MkdirAll(path, 0o777); err != nil {
				return fmt.Errorf("error in creating the directory path %s: %w", path, err)
			}
		}
	}
	return nil
}

// Helper function to create slice of file paths.
func createFile(filePaths ...string) error {
	for _, path := range filePaths {
		if path != "" {
			file, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("error in creating the file %s: %w", path, err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("error in closing the file %s: %w", path, err)
			}
		}
	}
	return nil
}

// Helper function to remove slice of directory paths.
func removeAllDirPaths(dirPaths ...string) error {
	for _, path := range dirPaths {
		if path != "" {
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("error in removing the directory path %s: %w", path, err)
			}
		}
	}
	return nil
}
