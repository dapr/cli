/*
Copyright 2023 The Dapr Authors
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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDashboardChart(t *testing.T) {
	testCases := []struct {
		runtimeVersion  string
		expectDashboard bool
		expectError     bool
	}{
		{
			runtimeVersion:  "1.9.6",
			expectDashboard: true,
			expectError:     false,
		},
		{
			runtimeVersion:  "1.10.7",
			expectDashboard: true,
			expectError:     false,
		},
		{
			runtimeVersion:  "1.10.99",
			expectDashboard: true,
			expectError:     false,
		},
		{
			runtimeVersion:  "1.11.0",
			expectDashboard: false,
			expectError:     false,
		},
		{
			runtimeVersion:  "1.11.0",
			expectDashboard: false,
			expectError:     false,
		},
		{
			runtimeVersion:  "1.12.7",
			expectDashboard: false,
			expectError:     false,
		},
		{
			runtimeVersion:  "Bad Version",
			expectDashboard: false,
			expectError:     true,
		},
	}
	for _, tc := range testCases {
		t.Run("Validating version "+tc.runtimeVersion, func(t *testing.T) {
			hasDashboard, err := IsDashboardIncluded(tc.runtimeVersion)
			if tc.expectError {
				assert.Error(t, err, "expected an error")
			} else {
				assert.NoError(t, err, "expected an error")
			}

			assert.Equal(t, tc.expectDashboard, hasDashboard, "dashboard expectation")
		})
	}
}
