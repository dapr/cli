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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/go-version"
	yaml "gopkg.in/yaml.v2"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
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

type helmChartItems struct {
	Entries struct {
		Dapr []struct {
			Version string `yaml:"appVersion"`
		}
		DaprDashboard []struct {
			Version string `yaml:"appVersion"`
		} `yaml:"dapr-dashboard"`
	}
}

func GetDashboardVersion() (string, error) {
	return GetLatestReleaseGithub(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", DaprGitHubOrg, DashboardGitHubRepo))
}

func GetDaprVersion() (string, error) {
	version, err := GetLatestReleaseGithub(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", DaprGitHubOrg, DaprGitHubRepo))
	if err != nil {
		print.WarningStatusEvent(os.Stdout, "Failed to get runtime version: '%s'. Trying secondary source", err)

		version, err = GetLatestReleaseHelmChart("https://dapr.github.io/helm-charts/index.yaml")
		if err != nil {
			return "", err
		}
	}

	return version, nil
}

func GetVersionFromURL(releaseURL string, parseVersion func(body []byte) (string, error)) (string, error) {
	req, err := http.NewRequest(http.MethodGet, releaseURL, nil)
	if err != nil {
		return "", err
	}

	githubToken := utils.GetEnv("GITHUB_TOKEN", "")
	if githubToken != "" {
		req.Header.Add("Authorization", "token "+githubToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s - %s", releaseURL, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return parseVersion(body)
}

// GetLatestReleaseGithub return the latest release version of dapr from GitHub API.
func GetLatestReleaseGithub(githubURL string) (string, error) {
	return GetVersionFromURL(githubURL, func(body []byte) (string, error) {
		var githubRepoReleases []githubRepoReleaseItem
		err := json.Unmarshal(body, &githubRepoReleases)
		if err != nil {
			return "", err
		}

		if len(githubRepoReleases) == 0 {
			return "", fmt.Errorf("no releases")
		}

		defaultVersion, _ := version.NewVersion("0.0.0")
		latestVersion := defaultVersion

		for _, release := range githubRepoReleases {
			if !strings.Contains(release.TagName, "-rc") {
				cur, _ := version.NewVersion(strings.TrimPrefix(release.TagName, "v"))
				if cur.GreaterThan(latestVersion) {
					latestVersion = cur
				}
			}
		}

		if latestVersion.Equal(defaultVersion) {
			return "", fmt.Errorf("no releases")
		}

		return latestVersion.String(), nil
	})
}

// GetLatestReleaseHelmChart return the latest release version of dapr from helm chart static index.yaml.
func GetLatestReleaseHelmChart(helmChartURL string) (string, error) {
	return GetVersionFromURL(helmChartURL, func(body []byte) (string, error) {
		var helmChartReleases helmChartItems
		err := yaml.Unmarshal(body, &helmChartReleases)
		if err != nil {
			return "", err
		}
		if len(helmChartReleases.Entries.Dapr) == 0 {
			return "", fmt.Errorf("no releases")
		}

		for _, release := range helmChartReleases.Entries.Dapr {
			if !strings.Contains(release.Version, "-rc") {
				return release.Version, nil
			}
		}

		// Did not find a non-rc version, so we fallback to an RC.
		// This is helpful to allow us to validate installation of new charts (Dashboard).
		for _, release := range helmChartReleases.Entries.Dapr {
			return release.Version, nil
		}

		return "", fmt.Errorf("no releases")
	})
}
