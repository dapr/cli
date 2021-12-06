// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHAMode(t *testing.T) {
	t.Parallel()
	t.Run("ha mode", func(t *testing.T) {
		t.Parallel()
		s := []StatusOutput{
			{
				Replicas: 3,
			},
		}

		r := highAvailabilityEnabled(s)
		assert.True(t, r)
	})

	t.Run("non-ha mode", func(t *testing.T) {
		t.Parallel()
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
	t.Parallel()
	val, err := upgradeChartValues("1", "2", "3", true, true, []string{})
	assert.NoError(t, err)
	assert.Len(t, val, 2)
}

func TestArgsChartValues(t *testing.T) {
	t.Parallel()
	val, err := upgradeChartValues("1", "2", "3", true, true, []string{"a=b", "b=c"})
	assert.NoError(t, err)
	assert.Len(t, val, 4)
}

func TestIsDowngrade(t *testing.T) {
	assert.True(t, isDowngrade("1.3.0", "1.4.0-rc.5"))
	assert.True(t, isDowngrade("1.3.0", "1.4.0"))
	assert.False(t, isDowngrade("1.4.0-rc.5", "1.3.0"))
	assert.False(t, isDowngrade("1.4.0", "1.3.0"))
}
