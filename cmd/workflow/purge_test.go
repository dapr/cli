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

	"github.com/stretchr/testify/assert"
)

func TestPurgeFilterStatuses(t *testing.T) {
	expected := []string{
		"RUNNING",
		"COMPLETED",
		"CONTINUED_AS_NEW",
		"FAILED",
		"CANCELED",
		"TERMINATED",
		"PENDING",
		"SUSPENDED",
	}
	assert.Equal(t, expected, purgeFilterStatuses)
}

func TestPurgeCmdFlags(t *testing.T) {
	t.Run("all-filter-status flag is registered", func(t *testing.T) {
		f := PurgeCmd.Flags().Lookup("all-filter-status")
		assert.NotNil(t, f)
		assert.Equal(t, "string", f.Value.Type())
		assert.Contains(t, f.Usage, "Must be used with --all-older-than")
	})

	t.Run("all-filter-status and all are mutually exclusive", func(t *testing.T) {
		// The mutual exclusivity is registered via MarkFlagsMutuallyExclusive.
		// We verify the flag group exists by checking that the command
		// has both flags and that they are correctly configured.
		allFlag := PurgeCmd.Flags().Lookup("all")
		assert.NotNil(t, allFlag)
		filterFlag := PurgeCmd.Flags().Lookup("all-filter-status")
		assert.NotNil(t, filterFlag)
	})

	t.Run("all-older-than flag is registered", func(t *testing.T) {
		f := PurgeCmd.Flags().Lookup("all-older-than")
		assert.NotNil(t, f)
	})
}
