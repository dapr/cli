/*
Copyright 2024 The Dapr Authors
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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersionedNumberOfPods(t *testing.T) {
	tests := []struct {
		name           string
		isHAEnabled    bool
		details        VersionDetails
		expectedNumber int
		expectedError  bool
	}{
		{
			name:           "HA enabled with latest version",
			isHAEnabled:    true,
			details:        VersionDetails{UseDaprLatestVersion: true},
			expectedNumber: numHAPodsWithScheduler,
			expectedError:  false,
		},
		{
			name:           "HA enabled with old version",
			isHAEnabled:    true,
			details:        VersionDetails{UseDaprLatestVersion: false, RuntimeVersion: "1.13.0"},
			expectedNumber: numHAPodsOld,
			expectedError:  false,
		},
		{
			name:           "HA disabled with latest version",
			isHAEnabled:    false,
			details:        VersionDetails{UseDaprLatestVersion: true},
			expectedNumber: numNonHAPodsWithHAScheduler,
			expectedError:  false,
		},
		{
			name:           "HA disabled with old version",
			isHAEnabled:    false,
			details:        VersionDetails{UseDaprLatestVersion: false, RuntimeVersion: "1.13.0"},
			expectedNumber: numNonHAPodsOld,
			expectedError:  false,
		},
		{
			name:           "HA enabled with new version",
			isHAEnabled:    true,
			details:        VersionDetails{UseDaprLatestVersion: false, RuntimeVersion: "1.14.4"},
			expectedNumber: numHAPodsWithScheduler,
			expectedError:  false,
		},
		{
			name:           "HA disabled with new version",
			isHAEnabled:    false,
			details:        VersionDetails{UseDaprLatestVersion: false, RuntimeVersion: "1.14.4"},
			expectedNumber: numNonHAPodsWithScheduler,
			expectedError:  false,
		},
		{
			name:           "HA enabled with invalid version",
			isHAEnabled:    true,
			details:        VersionDetails{UseDaprLatestVersion: false, RuntimeVersion: "invalid version"},
			expectedNumber: 0,
			expectedError:  true,
		},
		{
			name:           "HA disabled with invalid version",
			isHAEnabled:    false,
			details:        VersionDetails{UseDaprLatestVersion: false, RuntimeVersion: "invalid version"},
			expectedNumber: 0,
			expectedError:  true,
		},
		{
			name:           "HA enabled with new RC version",
			isHAEnabled:    true,
			details:        VersionDetails{UseDaprLatestVersion: false, RuntimeVersion: "1.15.0-rc.1"},
			expectedNumber: numHAPodsWithScheduler,
			expectedError:  false,
		},
		{
			name:           "HA disabled with new RC version",
			isHAEnabled:    false,
			details:        VersionDetails{UseDaprLatestVersion: false, RuntimeVersion: "1.15.0-rc.1"},
			expectedNumber: numNonHAPodsWithHAScheduler,
			expectedError:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			number, err := getVersionedNumberOfPods(tc.isHAEnabled, tc.details)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedNumber, number)
			}
		})
	}
}
