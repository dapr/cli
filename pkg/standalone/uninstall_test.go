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

package standalone

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControlPlaneContainersToRemove(t *testing.T) {
	tests := []struct {
		name          string
		binaries      []string
		wantPlacement bool
		wantScheduler bool
	}{
		{
			name:          "docker install has only daprd",
			binaries:      []string{daprRuntimeFilePrefix},
			wantPlacement: true,
			wantScheduler: true,
		},
		{
			name:          "slim install has placement and scheduler",
			binaries:      []string{daprRuntimeFilePrefix, placementServiceFilePrefix, schedulerServiceFilePrefix},
			wantPlacement: false,
			wantScheduler: false,
		},
		{
			// `dapr init --slim --scheduler-placement` installs no placement binary.
			// The dapr_placement container belongs to another installation.
			name:          "slim install with scheduler placement has only scheduler",
			binaries:      []string{daprRuntimeFilePrefix, schedulerServiceFilePrefix},
			wantPlacement: false,
			wantScheduler: false,
		},
		{
			// A runtime before 1.14 has no scheduler binary.
			name:          "slim install of a runtime without scheduler has only placement",
			binaries:      []string{daprRuntimeFilePrefix, placementServiceFilePrefix},
			wantPlacement: false,
			wantScheduler: false,
		},
		{
			name:          "empty bin folder",
			binaries:      nil,
			wantPlacement: true,
			wantScheduler: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			for _, b := range tc.binaries {
				require.NoError(t, os.WriteFile(binaryFilePathWithDir(binDir, b), []byte("bin"), 0o600))
			}

			placement, scheduler := controlPlaneContainersToRemove(binDir)
			assert.Equal(t, tc.wantPlacement, placement, "remove the placement container")
			assert.Equal(t, tc.wantScheduler, scheduler, "remove the scheduler container")
		})
	}

	t.Run("missing bin folder", func(t *testing.T) {
		placement, scheduler := controlPlaneContainersToRemove(filepath.Join(t.TempDir(), "does-not-exist"))
		assert.True(t, placement)
		assert.True(t, scheduler)
	})
}
