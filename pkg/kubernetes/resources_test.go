package kubernetes

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetResources(t *testing.T) {
	tests := []struct {
		name                  string
		folder                string
		expectError           bool
		expectedCount         int
		expectedResourceKinds []string
	}{
		{
			name:                  "resources from testdata",
			folder:                filepath.Join("testdata", "resources"),
			expectError:           false,
			expectedCount:         3,
			expectedResourceKinds: []string{"Component", "Configuration", "Resiliency"},
		},
		{
			name:        "non-existent folder",
			folder:      filepath.Join("testdata", "non-existent"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, err := getResources(tt.folder)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, resources, tt.expectedCount)
			foundKinds := []string{}
			for _, resource := range resources {
				foundKinds = append(foundKinds, resource.GetObjectKind().GroupVersionKind().Kind)
			}
			assert.ElementsMatch(t, tt.expectedResourceKinds, foundKinds)
		})
	}
}
