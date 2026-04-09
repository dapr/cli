/*
Copyright 2026 The Dapr Authors
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

	"github.com/dapr/cli/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPurgeFilterStatuses(t *testing.T) {
	assert.Equal(t, workflow.RuntimeStatuses, purgeFilterStatuses)
}

func TestPurgeCmdFlags(t *testing.T) {
	t.Run("all-filter-status flag is registered", func(t *testing.T) {
		f := PurgeCmd.Flags().Lookup("all-filter-status")
		assert.NotNil(t, f)
		assert.Equal(t, "string", f.Value.Type())
		assert.Contains(t, f.Usage, "Must be used with --all-older-than")
	})

	t.Run("all-filter-status and all are mutually exclusive", func(t *testing.T) {
		WorkflowCmd.SetArgs([]string{"purge", "--all", "--all-filter-status", "COMPLETED"})
		err := WorkflowCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "if any flags in the group [all-filter-status all] are set none of the others can be")
	})

	t.Run("all-older-than flag is registered", func(t *testing.T) {
		f := PurgeCmd.Flags().Lookup("all-older-than")
		assert.NotNil(t, f)
	})

	t.Run("non-terminal status without force errors", func(t *testing.T) {
		WorkflowCmd.SetArgs([]string{"purge", "--all-older-than", "1s", "--all-filter-status", "RUNNING"})
		err := WorkflowCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--force is required when using --all-filter-status with a non-terminal status")
	})
}
