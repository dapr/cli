// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package version

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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

// GetLatestRelease return the latest release version of dapr.
func GetLatestRelease(gitHubOrg, gitHubRepo string) (string, error) {
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", gitHubOrg, gitHubRepo)

	req, err := http.NewRequest("GET", releaseURL, nil)
	if err != nil {
		return "", fmt.Errorf("error: %w", err)
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken != "" {
		req.Header.Add("Authorization", "token "+githubToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s - %s", releaseURL, resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error: %w", err)
	}

	var githubRepoReleases []githubRepoReleaseItem
	err = json.Unmarshal(body, &githubRepoReleases)
	if err != nil {
		return "", fmt.Errorf("error: %w", err)
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
