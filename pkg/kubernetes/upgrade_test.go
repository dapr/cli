/*
Copyright 2021 The Dapr Authors
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

func TestHAMode(t *testing.T) {
	t.Run("ha mode", func(t *testing.T) {
		s := []StatusOutput{
			{
				Replicas: 3,
			},
		}

		r := highAvailabilityEnabled(s)
		assert.True(t, r)
	})

	t.Run("non-ha mode", func(t *testing.T) {
		s := []StatusOutput{
			{
				Replicas: 1,
			},
		}

		r := highAvailabilityEnabled(s)
		assert.False(t, r)
	})
}

func TestMTLSChartValues(t *testing.T) {
	args := []string{}
	mockUpgradeConfig := UpgradeConfig{
		RuntimeVersion:   "mocker_version_1.0.0",
		Args:             args,
		Timeout:          0,
		ImageRegistryURI: "",
	}

	val, err := upgradeChartValues("1", "2", "3", true, true, mockUpgradeConfig)
	assert.NoError(t, err)
	assert.Len(t, val, 2)
}

func TestArgsChartValues(t *testing.T) {
	args := []string{"a=b", "c=d"}
	mockUpgradeConfig := UpgradeConfig{
		RuntimeVersion:   "mocker_version_1.0.0",
		Args:             args,
		Timeout:          0,
		ImageRegistryURI: "",
	}
	val, err := upgradeChartValues("1", "2", "3", true, true, mockUpgradeConfig)
	assert.NoError(t, err)
	assert.Len(t, val, 4)
}

func TestIsDowngrade(t *testing.T) {
	assert.True(t, isDowngrade("1.3.0", "1.4.0-rc.5"))
	assert.True(t, isDowngrade("1.3.0", "1.4.0"))
	assert.False(t, isDowngrade("1.4.0-rc.5", "1.3.0"))
	assert.False(t, isDowngrade("1.4.0", "1.3.0"))
}
