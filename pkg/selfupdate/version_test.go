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

package selfupdate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func feed(tags ...string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><feed xmlns="http://www.w3.org/2005/Atom"><title>Release notes from cli</title>`)
	for _, t := range tags {
		sb.WriteString(`<entry><id>tag:github.com,2008:Repository/1/` + t + `</id><link rel="alternate" type="text/html" href="https://github.com/dapr/cli/releases/tag/` + t + `"/><title>` + t + `</title></entry>`)
	}
	sb.WriteString(`</feed>`)
	return sb.String()
}

func TestParseLatestStable(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"skips rc", feed("v1.18.3", "v1.19.0-rc.1", "v1.18.2"), "1.18.3", false},
		{"out of order", feed("v1.17.1", "v1.18.0"), "1.18.0", false},
		{"only rc", feed("v1.19.0-rc.1"), "", true},
		{"not xml", "not xml", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseLatestStable(strings.NewReader(tc.input))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	assert.Equal(t, "1.2.3", NormalizeVersion("v1.2.3"))
	assert.Equal(t, "1.2.3", NormalizeVersion("1.2.3"))
}

func TestDecide(t *testing.T) {
	tests := []struct {
		name            string
		current, target string
		explicit        bool
		want            Action
	}{
		{"older", "1.18.2", "1.18.3", false, ActionInstall},
		{"equal", "1.18.3", "1.18.3", false, ActionUpToDate},
		{"equal with v explicit", "v1.18.3", "1.18.3", true, ActionUpToDate},
		{"prerelease ahead", "1.19.0-rc.1", "1.18.3", false, ActionAheadOfLatest},
		{"explicit downgrade", "1.18.3", "1.17.0", true, ActionInstall},
		{"non-semver current", "edge", "1.18.3", false, ActionInstall},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Decide(tc.current, tc.target, tc.explicit))
		})
	}
}
