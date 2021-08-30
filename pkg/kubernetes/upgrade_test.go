// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//nolint
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

//nolint
func TestMTLSChartValues(t *testing.T) {
	val, err := upgradeChartValues("1", "2", "3", true, true, []string{})
	assert.NoError(t, err)
	assert.Len(t, val, 2)
}

//nolint
func TestArgsChartValues(t *testing.T) {
	val, err := upgradeChartValues("1", "2", "3", true, true, []string{"a=b", "b=c"})
	assert.NoError(t, err)
	assert.Len(t, val, 4)
}
