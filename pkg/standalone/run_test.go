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

package standalone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAppID(t *testing.T) {
	mockDaprMeta := &DaprMeta{}
	existingAppIDMap := make(map[string]bool, 1)
	existingAppIDMap["app1"] = true
	mockDaprMeta.ExistingIDs = existingAppIDMap

	basicConfigUniqueAppID := &RunConfig{AppID: "app1"}
	basicConfigEmptyAppID := &RunConfig{AppID: ""}

	testcases := []struct {
		name         string
		runConfig    *RunConfig
		mockDaprMeta *DaprMeta
		expectedErr  bool
	}{
		{
			name:         "unique app id",
			runConfig:    basicConfigUniqueAppID,
			mockDaprMeta: &DaprMeta{},
			expectedErr:  false,
		},
		{
			name:         "empty app id",
			runConfig:    basicConfigEmptyAppID,
			mockDaprMeta: &DaprMeta{},
			expectedErr:  false,
		},
		{
			name:         "provided app id already exists",
			runConfig:    basicConfigUniqueAppID,
			mockDaprMeta: mockDaprMeta,
			expectedErr:  true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.runConfig.validateAppID(tc.mockDaprMeta)
			assert.Equal(t, tc.expectedErr, err != nil)
		})
	}
}
