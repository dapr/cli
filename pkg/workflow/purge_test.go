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

	"github.com/dapr/kit/ptr"
	"github.com/stretchr/testify/assert"
)

func TestBuildPurgeFilter(t *testing.T) {
	t.Run("nil status uses terminal filter", func(t *testing.T) {
		filter := BuildPurgeFilter(nil)
		assert.True(t, filter.Terminal)
		assert.Nil(t, filter.Status)
	})

	t.Run("with status uses status filter instead of terminal", func(t *testing.T) {
		filter := BuildPurgeFilter(ptr.Of("FAILED"))
		assert.False(t, filter.Terminal)
		assert.NotNil(t, filter.Status)
		assert.Equal(t, "FAILED", *filter.Status)
	})

	t.Run("with COMPLETED status", func(t *testing.T) {
		filter := BuildPurgeFilter(ptr.Of("COMPLETED"))
		assert.False(t, filter.Terminal)
		assert.Equal(t, "COMPLETED", *filter.Status)
	})

	t.Run("with RUNNING status", func(t *testing.T) {
		filter := BuildPurgeFilter(ptr.Of("RUNNING"))
		assert.False(t, filter.Terminal)
		assert.Equal(t, "RUNNING", *filter.Status)
	})
}
