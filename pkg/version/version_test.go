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

package version

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	v1 "github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pushEmptyImage pushes a minimal image with the given tag to the test registry.
func pushEmptyImage(t *testing.T, ref string) {
	t.Helper()
	img := mutate.MediaType(empty.Image, v1.DockerManifestSchema2)
	err := crane.Push(img, ref)
	require.NoError(t, err, "failed to push image %s", ref)
}

// startTestRegistry starts an in-memory OCI registry and returns the
// host:port string (without scheme) and a cleanup function.
func startTestRegistry(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(registry.New())
	t.Cleanup(srv.Close)
	// httptest URLs are http://host:port — strip the scheme for use as a registry.
	return strings.TrimPrefix(srv.URL, "http://")
}

func TestGetLatestVersion(t *testing.T) {
	tests := []struct {
		name        string
		tags        []string
		expectedVer string
		expectedErr string
	}{
		{
			name:        "RC releases are skipped",
			tags:        []string{"v1.2.3-rc.1", "v1.2.2"},
			expectedVer: "1.2.2",
		},
		{
			name:        "Only latest version is returned",
			tags:        []string{"v1.4.4", "v1.5.1"},
			expectedVer: "1.5.1",
		},
		{
			name:        "Only latest stable version is returned",
			tags:        []string{"v1.5.2-rc.1", "v1.4.4", "v1.5.1"},
			expectedVer: "1.5.1",
		},
		{
			name:        "Tags without v prefix work",
			tags:        []string{"1.4.4", "1.5.1"},
			expectedVer: "1.5.1",
		},
		{
			name:        "Only RCs returns error",
			tags:        []string{"v1.2.3-rc.1"},
			expectedErr: "no stable releases found",
		},
		{
			name:        "Malformed version tags are skipped",
			tags:        []string{"vedge", "latest", "v1.5.1"},
			expectedVer: "1.5.1",
		},
		{
			name:        "Only malformed tags returns error",
			tags:        []string{"vedge", "latest"},
			expectedErr: "no stable releases found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host := startTestRegistry(t)
			imageRef := fmt.Sprintf("%s/dapr/dapr", host)

			for _, tag := range tc.tags {
				pushEmptyImage(t, fmt.Sprintf("%s:%s", imageRef, tag))
			}

			ver, err := GetLatestVersion(imageRef)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				assert.Empty(t, ver)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedVer, ver)
			}
		})
	}
}

func TestGetLatestVersionNoTags(t *testing.T) {
	host := startTestRegistry(t)
	imageRef := fmt.Sprintf("%s/dapr/nonexistent", host)

	ver, err := GetLatestVersion(imageRef)
	assert.Error(t, err)
	assert.Empty(t, ver)
}

func TestGetLatestVersionInvalidRef(t *testing.T) {
	ver, err := GetLatestVersion("://invalid")
	assert.Error(t, err)
	assert.Empty(t, ver)
}

func TestGetLatestVersionBadAddress(t *testing.T) {
	ver, err := GetLatestVersion("a.super.non.existent.domain/dapr/dapr")
	assert.Error(t, err)
	assert.Empty(t, ver)
}

func TestDaprImageRef(t *testing.T) {
	tests := []struct {
		name             string
		imageRegistryURL string
		expected         string
	}{
		{
			name:             "custom registry",
			imageRegistryURL: "localhost:5000",
			expected:         "localhost:5000/dapr/dapr",
		},
		{
			name:     "default Docker Hub",
			expected: DaprDefaultImage,
		},
		{
			name:             "default Docker Hub, empty registry URL",
			imageRegistryURL: "",
			expected:         DaprDefaultImage,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := DaprImageRef(tc.imageRegistryURL)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDashboardImageRef(t *testing.T) {
	tests := []struct {
		name             string
		imageRegistryURL string
		expected         string
	}{
		{
			name:             "custom registry",
			imageRegistryURL: "localhost:5000",
			expected:         "localhost:5000/dapr/dashboard",
		},
		{
			name:     "default Docker Hub",
			expected: DashboardDefaultImage,
		},
		{
			name:             "default Docker Hub, empty registry URL",
			imageRegistryURL: "",
			expected:         DashboardDefaultImage,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := DashboardImageRef(tc.imageRegistryURL)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestImageRefCanBeResolved(t *testing.T) {
	// Verify that the default image refs are valid repository references.
	refs := []string{DaprDefaultImage, DashboardDefaultImage}
	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			_, err := name.NewRepository(ref)
			require.NoError(t, err, "default image ref %q should be a valid repository", ref)
		})
	}
}
