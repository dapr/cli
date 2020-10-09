// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package version

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	// DaprGitHubOrg is the org name of dapr on GitHub.
	DaprGitHubOrg = "dapr"
	// DaprGitHubRepo is the repo name of dapr runtime on GitHub.
	DaprGitHubRepo = "dapr"
	// DashboardGitHubRepo is the repo name of dapr dashboard on GitHub.
	DashboardGitHubRepo = "dashboard"
)

type githubRepoReleaseItem struct {
	URL     string `json:"url"`
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Draft   bool   `json:"draft"`
}

// nolint:gosec
// GetLatestRelease return the latest release version of dapr.
func GetLatestRelease(gitHubOrg, gitHubRepo string) (string, error) {
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", gitHubOrg, gitHubRepo)
	resp, err := http.Get(releaseURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s - %s", releaseURL, resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var githubRepoReleases []githubRepoReleaseItem
	err = json.Unmarshal(body, &githubRepoReleases)
	if err != nil {
		return "", err
	}

	if len(githubRepoReleases) == 0 {
		return "", fmt.Errorf("no releases")
	}

	for _, release := range githubRepoReleases {
		if !strings.Contains(release.TagName, "-rc") {
			return release.TagName, nil
		}
	}

	return "", fmt.Errorf("no releases")
}
