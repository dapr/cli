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
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetVersionsGithub(t *testing.T) {
	// Ensure a clean environment.

	tests := []struct {
		Name         string
		Path         string
		ResponseBody string
		ExpectedErr  string
		ExpectedVer  string
	}{
		{
			"RC releases are skipped",
			"/no_rc",
			`[
  {
    "url": "https://api.github.com/repos/dapr/dapr/releases/44766923",
    "html_url": "https://github.com/dapr/dapr/releases/tag/v1.2.3-rc.1",
    "id": 44766926,
    "tag_name": "v1.2.3-rc.1",
    "target_commitish": "master",
    "name": "Dapr Runtime v1.2.3-rc.1",
    "draft": false,
    "prerelease": false
  },
  {
    "url": "https://api.github.com/repos/dapr/dapr/releases/44766923",
    "html_url": "https://github.com/dapr/dapr/releases/tag/v1.2.2",
    "id": 44766923,
    "tag_name": "v1.2.2",
    "target_commitish": "master",
    "name": "Dapr Runtime v1.2.2",
    "draft": false,
    "prerelease": false
  }
]
			`,
			"",
			"1.2.2",
		},
		{
			"Only latest version is got",
			"/latest",
			`[
  {
    "url": "https://api.github.com/repos/dapr/dapr/releases/44766923",
    "html_url": "https://github.com/dapr/dapr/releases/tag/v1.4.4",
    "id": 44766926,
    "tag_name": "v1.4.4",
    "target_commitish": "master",
    "name": "Dapr Runtime v1.4.4",
    "draft": false,
    "prerelease": false
  },
  {
    "url": "https://api.github.com/repos/dapr/dapr/releases/44766923",
    "html_url": "https://github.com/dapr/dapr/releases/tag/v1.5.1",
    "id": 44766923,
    "tag_name": "v1.5.1",
    "target_commitish": "master",
    "name": "Dapr Runtime v1.5.1",
    "draft": false,
    "prerelease": false
  }
]
			`,
			"",
			"1.5.1",
		},
		{
			"Only latest stable version is got",
			"/latest_stable",
			`[
  {
    "url": "https://api.github.com/repos/dapr/dapr/releases/44766923",
    "html_url": "https://github.com/dapr/dapr/releases/tag/v1.5.2-rc.1",
    "id": 44766926,
    "tag_name": "v1.5.2-rc.1",
    "target_commitish": "master",
    "name": "Dapr Runtime v1.5.2-rc.1",
    "draft": false,
    "prerelease": true
  },
  {
    "url": "https://api.github.com/repos/dapr/dapr/releases/44766923",
    "html_url": "https://github.com/dapr/dapr/releases/tag/v1.4.4",
    "id": 44766926,
    "tag_name": "v1.4.4",
    "target_commitish": "master",
    "name": "Dapr Runtime v1.4.4",
    "draft": false,
    "prerelease": false
  },
  {
    "url": "https://api.github.com/repos/dapr/dapr/releases/44766923",
    "html_url": "https://github.com/dapr/dapr/releases/tag/v1.5.1",
    "id": 44766923,
    "tag_name": "v1.5.1",
    "target_commitish": "master",
    "name": "Dapr Runtime v1.5.1",
    "draft": false,
    "prerelease": false
  }
]
			`,
			"",
			"1.5.1",
		},
		{
			"Malformed JSON",
			"/malformed",
			"[",
			"unexpected end of JSON input",
			"",
		},
		{
			"Only RCs",
			"/only_rcs",
			`[
  {
    "url": "https://api.github.com/repos/dapr/dapr/releases/44766923",
    "html_url": "https://github.com/dapr/dapr/releases/tag/v1.2.3-rc.1",
    "id": 44766926,
    "tag_name": "v1.2.3-rc.1",
    "target_commitish": "master",
    "name": "Dapr Runtime v1.2.3-rc.1",
    "draft": false,
    "prerelease": false
  }
]			`,
			"no releases",
			"",
		},
		{
			"Empty json",
			"/empty",
			"[]",
			"no releases",
			"",
		},
	}
	m := http.NewServeMux()
	s := http.Server{Addr: ":12345", Handler: m, ReadHeaderTimeout: time.Duration(5) * time.Second}

	for _, tc := range tests {
		body := tc.ResponseBody
		m.HandleFunc(tc.Path, func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, body)
		})
	}

	go func() {
		s.ListenAndServe()
	}()

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			version, err := GetLatestReleaseGithub(fmt.Sprintf("http://localhost:12345%s", tc.Path))
			assert.Equal(t, tc.ExpectedVer, version)
			if tc.ExpectedErr != "" {
				assert.EqualError(t, err, tc.ExpectedErr)
			}
		})
	}

	t.Run("error on 404", func(t *testing.T) {
		version, err := GetLatestReleaseGithub("http://localhost:12345/non-existant/path")
		assert.Equal(t, "", version)
		assert.EqualError(t, err, "http://localhost:12345/non-existant/path - 404 Not Found")
	})

	t.Run("error on bad addr", func(t *testing.T) {
		version, err := GetLatestReleaseGithub("http://a.super.non.existant.domain/")
		assert.Equal(t, "", version)
		assert.Error(t, err)
	})

	s.Shutdown(context.Background())
}

func TestGetVersionsHelm(t *testing.T) {
	// Ensure a clean environment.

	tests := []struct {
		Name         string
		Path         string
		ResponseBody string
		ExpectedErr  string
		ExpectedVer  string
	}{
		{
			"Use RC releases if there isn't a full release yet",
			"/fallback_to_rc",
			`apiVersion: v1
entries:
  dapr:
  - apiVersion: v1
    appVersion: 1.2.3-rc.1
    created: "2021-06-17T03:13:24.179849371Z"
    description: A Helm chart for Dapr on Kubernetes
    digest: 60d8d17b58ca316cdcbdb8529cf9ba2c9e2e0834383c677cafbf99add86ee7a0
    name: dapr
    urls:
    - https://dapr.github.io/helm-charts/dapr-1.2.3-rc.1.tgz
    version: 1.2.3-rc.1
  - apiVersion: v1
    appVersion: 1.2.2
    created: "2021-06-17T03:13:24.179849371Z"
    description: A Helm chart for Dapr on Kubernetes
    digest: 60d8d17b58ca316cdcbdb8529cf9ba2c9e2e0834383c677cafbf99add86ee7a0
    name: dapr
    urls:
    - https://dapr.github.io/helm-charts/dapr-1.2.2.tgz
    version: 1.2.2      `,
			"",
			"1.2.2",
		},
		{
			"Malformed YAML",
			"/malformed",
			"[",
			"yaml: line 1: did not find expected node content",
			"",
		},
		{
			"Empty YAML",
			"/empty",
			"",
			"no releases",
			"",
		},
		{
			"Only RCs",
			"/only_rcs",
			`apiVersion: v1
entries:
  dapr:
  - apiVersion: v1
    appVersion: 1.2.3-rc.1
    created: "2021-06-17T03:13:24.179849371Z"
    description: A Helm chart for Dapr on Kubernetes
    digest: 60d8d17b58ca316cdcbdb8529cf9ba2c9e2e0834383c677cafbf99add86ee7a0
    name: dapr
    urls:
    - https://dapr.github.io/helm-charts/dapr-1.2.3-rc.1.tgz
    version: 1.2.3-rc.1 `,
			"",
			"1.2.3-rc.1",
		},
	}
	m := http.NewServeMux()
	s := http.Server{Addr: ":12346", Handler: m, ReadHeaderTimeout: time.Duration(5) * time.Second}

	for _, tc := range tests {
		body := tc.ResponseBody
		m.HandleFunc(tc.Path, func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, body)
		})
	}

	go func() {
		s.ListenAndServe()
	}()

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			version, err := GetLatestReleaseHelmChart(fmt.Sprintf("http://localhost:12346%s", tc.Path))
			assert.Equal(t, tc.ExpectedVer, version)
			if tc.ExpectedErr != "" {
				assert.EqualError(t, err, tc.ExpectedErr)
			}
		})
	}

	s.Shutdown(context.Background())
}
