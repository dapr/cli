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
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Masterminds/semver/v3"

	"github.com/dapr/cli/pkg/download"
)

const (
	defaultReleaseBaseURL = "https://github.com/dapr/cli/releases"
	feedTimeout           = 30 * time.Second
)

// Updater replaces the running Dapr CLI binary with a released version.
type Updater struct {
	ReleaseBaseURL string                     // default "https://github.com/dapr/cli/releases"
	CurrentVersion string                     // version of the running CLI
	ExecutablePath string                     // default os.Executable; symlinks are always resolved
	GOOS, GOARCH   string                     // default runtime values
	DetectInstall  func(string) InstallMethod // default DetectInstallMethod
}

// Result describes the outcome of Run.
type Result struct {
	Action      Action
	From, To    string
	PackageHint string // non-empty when a package manager owns the install; nothing changed
}

// Run updates the CLI to requestedVersion, or to the latest stable release when
// requestedVersion is empty.
func (u *Updater) Run(ctx context.Context, requestedVersion string) (Result, error) {
	if requestedVersion != "" {
		if _, err := semver.StrictNewVersion(NormalizeVersion(requestedVersion)); err != nil {
			return Result{}, fmt.Errorf("invalid version %q: use a full version such as 1.18.3", requestedVersion)
		}
	}

	baseURL := valueOr(u.ReleaseBaseURL, defaultReleaseBaseURL)
	goos := valueOr(u.GOOS, runtime.GOOS)
	goarch := valueOr(u.GOARCH, runtime.GOARCH)
	detect := u.DetectInstall
	if detect == nil {
		detect = DetectInstallMethod
	}

	exePath, err := u.executablePath()
	if err != nil {
		return Result{}, err
	}
	CleanupOld(exePath)

	res := Result{From: u.CurrentVersion}
	if hint := PackageManagerHint(detect(exePath), requestedVersion, u.CurrentVersion); hint != "" {
		res.PackageHint = hint
		return res, nil
	}

	target := NormalizeVersion(requestedVersion)
	if target == "" {
		if target, err = latestStable(ctx, baseURL); err != nil {
			return Result{}, fmt.Errorf("failed to check for the latest Dapr CLI version: %w", err)
		}
	}
	res.To = target
	res.Action = Decide(u.CurrentVersion, target, requestedVersion != "")
	if res.Action != ActionInstall {
		return res, nil
	}

	if err := install(baseURL, target, goos, goarch, exePath); err != nil {
		return Result{}, err
	}
	return res, nil
}

func (u *Updater) executablePath() (string, error) {
	p := u.ExecutablePath
	if p == "" {
		var err error
		if p, err = os.Executable(); err != nil {
			return "", fmt.Errorf("failed to locate the Dapr CLI binary: %w", err)
		}
	}
	return resolveExecutable(p)
}

// resolveExecutable follows symlinks so the real binary is replaced (and the
// symlink left intact) and install detection sees the real location.
func resolveExecutable(p string) (string, error) {
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", fmt.Errorf("failed to resolve the Dapr CLI binary path: %w", err)
	}
	return resolved, nil
}

func latestStable(ctx context.Context, baseURL string) (string, error) {
	client := &http.Client{
		Timeout:   feedTimeout,
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+".atom", nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from %s", resp.StatusCode, req.URL)
	}
	return ParseLatestStable(resp.Body)
}

func install(baseURL, version, goos, goarch, exePath string) error {
	tmp, err := os.MkdirTemp("", "dapr-update-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmp)

	asset := fmt.Sprintf("dapr_%s_%s.%s", goos, goarch, download.ArchiveExt())
	assetURL := fmt.Sprintf("%s/download/v%s/%s", baseURL, version, asset)

	archive, err := download.File(tmp, assetURL)
	if err != nil {
		return fmt.Errorf("failed to download Dapr CLI: version %s not found or not available for %s/%s: %w", version, goos, goarch, err)
	}
	checksum, err := download.File(tmp, assetURL+".sha256")
	if err != nil {
		return fmt.Errorf("checksum file not published for Dapr CLI version %s; cannot verify the download: %w", version, err)
	}
	if err = VerifyChecksum(archive, checksum); err != nil {
		return err
	}

	extracted, err := download.ExtractBinary(archive, tmp, "dapr")
	if err != nil {
		return err
	}
	if extracted == "" {
		return fmt.Errorf("dapr binary not found in archive %s", asset)
	}

	staged, err := StageBinary(exePath, extracted)
	if err != nil {
		return err
	}
	return ReplaceBinary(exePath, staged, goos)
}

func valueOr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
