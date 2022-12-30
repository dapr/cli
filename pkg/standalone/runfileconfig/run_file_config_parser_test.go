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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	validRunFilePath    = filepath.Join("..", "testdata", "runfileconfig", "test_run_config.yaml")
	invalidRunFilePath1 = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_invalid_path.yaml")
	invalidRunFilePath2 = filepath.Join("..", "testdata", "runfileconfig", "test_run_config_empty_app_dir.yaml")
)

func TestRunConfigParser(t *testing.T) {
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
}

func TestValidateRunConfig(t *testing.T) {
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
}

func TestGetApps(t *testing.T) {
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
			input:         "/tmp/test",
			expectedErr:   false,
			expectedAppID: "test",
		},
		{
			name:          "invalid absolute path",
			input:         "../test/",
			expectedErr:   true,
			expectedAppID: "",
		},
		{
			name:          "invalid absolute path",
			input:         "./test/",
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
