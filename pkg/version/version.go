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
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	goversion "github.com/hashicorp/go-version"
)

const (
	// DaprGitHubOrg is the org name of dapr on GitHub.
	DaprGitHubOrg = "dapr"
	// DaprGitHubRepo is the repo name of dapr runtime on GitHub.
	DaprGitHubRepo = "dapr"
	// DashboardGitHubRepo is the repo name of dapr dashboard on GitHub.
	DashboardGitHubRepo = "dashboard"

	// Default container image references used for version resolution.
	DaprDefaultImage      = "daprio/dapr"
	DashboardDefaultImage = "daprio/dashboard"
)

// GetLatestVersion lists tags on the given container image reference
// and returns the highest semver, non-prerelease version.
func GetLatestVersion(imageRef string) (string, error) {
	repo, err := name.NewRepository(imageRef)
	if err != nil {
		return "", fmt.Errorf("invalid image reference %q: %w", imageRef, err)
	}

	tags, err := remote.List(repo, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return "", fmt.Errorf("failed to list tags for %q: %w", imageRef, err)
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found for %q", imageRef)
	}

	defaultVersion, _ := goversion.NewVersion("0.0.0")
	latestVersion := defaultVersion

	for _, tag := range tags {
		cur, err := goversion.NewVersion(strings.TrimPrefix(tag, "v"))
		if err != nil || cur == nil {
			continue
		}
		if cur.Prerelease() != "" || cur.Metadata() != "" {
			continue
		}
		if cur.GreaterThan(latestVersion) {
			latestVersion = cur
		}
	}

	if latestVersion.Equal(defaultVersion) {
		return "", fmt.Errorf("no stable releases found for %q", imageRef)
	}

	return latestVersion.String(), nil
}

// DaprImageRef returns the full image reference for the Dapr runtime image
// based on the provided custom registry URL. If empty, it falls back to
// the default Docker Hub image.
func DaprImageRef(imageRegistryURL string) string {
	if imageRegistryURL != "" {
		return imageRegistryURL + "/dapr/dapr"
	}
	return DaprDefaultImage
}

// DashboardImageRef returns the full image reference for the Dapr dashboard
// image based on the provided custom registry URL. If empty, it falls back
// to the default Docker Hub image.
func DashboardImageRef(imageRegistryURL string) string {
	if imageRegistryURL != "" {
		return imageRegistryURL + "/dapr/dashboard"
	}
	return DashboardDefaultImage
}
