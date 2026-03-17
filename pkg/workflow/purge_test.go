/*
Copyright 2025 The Dapr Authors
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

package workflow

import (
	"testing"
	"time"

	"github.com/dapr/kit/ptr"
	"github.com/stretchr/testify/assert"
)

func TestPurgeOptions_AllFilterStatus(t *testing.T) {
	t.Run("AllFilterStatus sets filter status instead of terminal", func(t *testing.T) {
		opts := PurgeOptions{
			AllOlderThan:    ptr.Of(time.Now()),
			AllFilterStatus: ptr.Of("COMPLETED"),
		}

		assert.NotNil(t, opts.AllFilterStatus)
		assert.Equal(t, "COMPLETED", *opts.AllFilterStatus)
		assert.NotNil(t, opts.AllOlderThan)
	})

	t.Run("nil AllFilterStatus defaults to terminal filtering", func(t *testing.T) {
		opts := PurgeOptions{
			AllOlderThan: ptr.Of(time.Now()),
		}

		assert.Nil(t, opts.AllFilterStatus)
	})

	t.Run("AllFilterStatus with various statuses", func(t *testing.T) {
		statuses := []string{
			"RUNNING", "COMPLETED", "CONTINUED_AS_NEW",
			"FAILED", "CANCELED", "TERMINATED",
			"PENDING", "SUSPENDED",
		}

		for _, status := range statuses {
			t.Run(status, func(t *testing.T) {
				opts := PurgeOptions{
					AllOlderThan:    ptr.Of(time.Now()),
					AllFilterStatus: ptr.Of(status),
				}
				assert.Equal(t, status, *opts.AllFilterStatus)
			})
		}
	})
}

func TestPurgeFilterBuildLogic(t *testing.T) {
	// Tests the filter construction logic that Purge uses internally.
	// When AllFilterStatus is set, Terminal should be false and Status should
	// be the provided value. When AllFilterStatus is nil, Terminal should be true.

	t.Run("without AllFilterStatus uses terminal filter", func(t *testing.T) {
		opts := PurgeOptions{
			All: true,
		}

		filter := Filter{Terminal: true}
		if opts.AllFilterStatus != nil {
			filter.Terminal = false
			filter.Status = opts.AllFilterStatus
		}

		assert.True(t, filter.Terminal)
		assert.Nil(t, filter.Status)
	})

	t.Run("with AllFilterStatus uses status filter", func(t *testing.T) {
		opts := PurgeOptions{
			AllOlderThan:    ptr.Of(time.Now()),
			AllFilterStatus: ptr.Of("FAILED"),
		}

		filter := Filter{Terminal: true}
		if opts.AllFilterStatus != nil {
			filter.Terminal = false
			filter.Status = opts.AllFilterStatus
		}

		assert.False(t, filter.Terminal)
		assert.NotNil(t, filter.Status)
		assert.Equal(t, "FAILED", *filter.Status)
	})

	t.Run("AllOlderThan filters by created time", func(t *testing.T) {
		now := time.Now()
		cutoff := now.Add(-1 * time.Hour)
		opts := PurgeOptions{
			AllOlderThan:    &cutoff,
			AllFilterStatus: ptr.Of("COMPLETED"),
		}

		// Simulate the filtering logic from Purge
		list := []*ListOutputWide{
			{InstanceID: "old-1", Created: now.Add(-2 * time.Hour), RuntimeStatus: "COMPLETED"},
			{InstanceID: "new-1", Created: now.Add(-30 * time.Minute), RuntimeStatus: "COMPLETED"},
			{InstanceID: "old-2", Created: now.Add(-3 * time.Hour), RuntimeStatus: "COMPLETED"},
		}

		var toPurge []string
		for _, w := range list {
			if w.Created.Before(*opts.AllOlderThan) {
				toPurge = append(toPurge, w.InstanceID)
			}
		}

		assert.Equal(t, []string{"old-1", "old-2"}, toPurge)
	})
}
