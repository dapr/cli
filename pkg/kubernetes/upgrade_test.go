// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

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
	val, err := mtlsChartValues("1", "2", "3", true)
	assert.NoError(t, err)
	assert.Len(t, val, 2)
}
