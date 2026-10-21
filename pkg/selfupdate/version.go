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
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/Masterminds/semver/v3"
)

type atomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Entries []atomEntry `xml:"http://www.w3.org/2005/Atom entry"`
}

type atomEntry struct {
	Links []atomLink `xml:"http://www.w3.org/2005/Atom link"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
}

// Action is the outcome of comparing the current version with the target.
type Action int

const (
	// ActionInstall means the target version should be installed.
	ActionInstall Action = iota
	// ActionUpToDate means the current version equals the target.
	ActionUpToDate
	// ActionAheadOfLatest means the current version is newer than the target.
	ActionAheadOfLatest
)

// ParseLatestStable returns the highest stable semver (without leading "v")
// found in a GitHub releases Atom feed.
func ParseLatestStable(feed io.Reader) (string, error) {
	var f atomFeed
	if err := xml.NewDecoder(feed).Decode(&f); err != nil {
		return "", fmt.Errorf("failed to parse releases feed: %w", err)
	}

	var latest *semver.Version
	for _, e := range f.Entries {
		for _, l := range e.Links {
			tag := path.Base(strings.TrimRight(l.Href, "/"))
			if strings.Contains(tag, "rc") {
				continue
			}
			v, err := semver.NewVersion(NormalizeVersion(tag))
			if err != nil {
				continue
			}
			if latest == nil || v.GreaterThan(latest) {
				latest = v
			}
		}
	}
	if latest == nil {
		return "", errors.New("no stable release found in releases feed")
	}
	return latest.String(), nil
}

// NormalizeVersion strips a single leading "v".
func NormalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// Decide determines what to do given the current and target versions.
func Decide(current, target string, explicit bool) Action {
	cur, err := semver.NewVersion(NormalizeVersion(current))
	if err != nil {
		return ActionInstall
	}
	tgt, err := semver.NewVersion(NormalizeVersion(target))
	if err != nil {
		return ActionInstall
	}
	if cur.Equal(tgt) {
		return ActionUpToDate
	}
	if explicit {
		return ActionInstall
	}
	if cur.GreaterThan(tgt) {
		return ActionAheadOfLatest
	}
	return ActionInstall
}
