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
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapr/cli/pkg/download"
)

// buildArchive returns a native-format release archive containing the dapr binary with content.
func buildArchive(t *testing.T, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	if download.ArchiveExt() == "zip" {
		zw := zip.NewWriter(&buf)
		w, err := zw.Create("dapr.exe")
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
		require.NoError(t, zw.Close())
		return buf.Bytes()
	}
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "dapr", Mode: 0o755, Size: int64(len(content))}))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

type releaseServer struct {
	*httptest.Server
	assetRequests atomic.Int32
}

type serverOpts struct {
	latest         string
	missingAsset   bool
	missingSHA     bool
	wrongChecksum  bool
	archiveContent string
}

func assetName() string {
	return "dapr_" + runtime.GOOS + "_" + runtime.GOARCH + "." + download.ArchiveExt()
}

func newReleaseServer(t *testing.T, o serverOpts) *releaseServer {
	t.Helper()
	if o.archiveContent == "" {
		o.archiveContent = "new"
	}
	archive := buildArchive(t, o.archiveContent)
	sum := sha256.Sum256(archive)
	sumHex := hex.EncodeToString(sum[:])
	if o.wrongChecksum {
		sumHex = strings.Repeat("0", 64)
	}
	asset := assetName()

	rs := &releaseServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/releases.atom", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(feed("v"+o.latest, "v1.19.0-rc.1", "v1.17.0")))
	})
	mux.HandleFunc("/releases/download/", func(w http.ResponseWriter, r *http.Request) {
		rs.assetRequests.Add(1)
		switch {
		case strings.HasSuffix(r.URL.Path, "/"+asset):
			if o.missingAsset {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(archive)
		case strings.HasSuffix(r.URL.Path, "/"+asset+".sha256"):
			if o.missingSHA {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(sumHex + " *" + asset + "\n"))
		default:
			http.NotFound(w, r)
		}
	})
	rs.Server = httptest.NewServer(mux)
	t.Cleanup(rs.Close)
	return rs
}

func newExe(t *testing.T) string {
	t.Helper()
	name := "dapr"
	if runtime.GOOS == "windows" {
		name = "dapr.exe"
	}
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, []byte("old"), 0o755)) //nolint:gosec
	return p
}

func newUpdater(baseURL, current, exe string) *Updater {
	return &Updater{
		ReleaseBaseURL: baseURL,
		CurrentVersion: current,
		ExecutablePath: exe,
		DetectInstall:  func(string) InstallMethod { return InstallDirect },
	}
}

func closedServerURL() string {
	s := httptest.NewServer(http.NotFoundHandler())
	u := s.URL
	s.Close()
	return u + "/releases"
}

func TestRun_UpdatesToLatest(t *testing.T) {
	srv := newReleaseServer(t, serverOpts{latest: "1.18.3"})
	exe := newExe(t)
	res, err := newUpdater(srv.URL+"/releases", "1.18.2", exe).Run(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, Result{Action: ActionInstall, From: "1.18.2", To: "1.18.3", PackageHint: ""}, res)
	assert.Equal(t, "new", readFile(t, exe))
}

func TestRun_AlreadyUpToDate(t *testing.T) {
	srv := newReleaseServer(t, serverOpts{latest: "1.18.3"})
	exe := newExe(t)
	res, err := newUpdater(srv.URL+"/releases", "1.18.3", exe).Run(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, ActionUpToDate, res.Action)
	assert.Equal(t, "old", readFile(t, exe))
	assert.Equal(t, int32(0), srv.assetRequests.Load())
}

func TestRun_AheadOfLatest(t *testing.T) {
	srv := newReleaseServer(t, serverOpts{latest: "1.18.3"})
	exe := newExe(t)
	res, err := newUpdater(srv.URL+"/releases", "1.19.0-rc.1", exe).Run(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, ActionAheadOfLatest, res.Action)
	assert.Equal(t, "old", readFile(t, exe))
	assert.Equal(t, int32(0), srv.assetRequests.Load())
}

func TestRun_ExplicitDowngrade(t *testing.T) {
	srv := newReleaseServer(t, serverOpts{latest: "1.18.3"})
	exe := newExe(t)
	res, err := newUpdater(srv.URL+"/releases", "1.18.3", exe).Run(context.Background(), "v1.17.0")
	require.NoError(t, err)
	assert.Equal(t, ActionInstall, res.Action)
	assert.Equal(t, "1.18.3", res.From)
	assert.Equal(t, "1.17.0", res.To)
	assert.Equal(t, "new", readFile(t, exe))
}

func TestRun_VersionNotFound(t *testing.T) {
	srv := newReleaseServer(t, serverOpts{latest: "1.18.3", missingAsset: true})
	exe := newExe(t)
	_, err := newUpdater(srv.URL+"/releases", "1.18.2", exe).Run(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), runtime.GOOS+"/"+runtime.GOARCH)
	assert.Equal(t, "old", readFile(t, exe))
}

func TestRun_InvalidRequestedVersion(t *testing.T) {
	for _, v := range []string{"latest", "1.18"} {
		t.Run(v, func(t *testing.T) {
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				http.NotFound(w, r)
			}))
			t.Cleanup(srv.Close)
			exe := newExe(t)
			_, err := newUpdater(srv.URL+"/releases", "1.18.2", exe).Run(context.Background(), v)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid version")
			assert.Contains(t, err.Error(), "1.18.3")
			assert.Zero(t, requests.Load())
			assert.Equal(t, "old", readFile(t, exe))
		})
	}
}

func TestRun_ChecksumMismatch(t *testing.T) {
	srv := newReleaseServer(t, serverOpts{latest: "1.18.3", wrongChecksum: true})
	exe := newExe(t)
	_, err := newUpdater(srv.URL+"/releases", "1.18.2", exe).Run(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
	assert.Equal(t, "old", readFile(t, exe))
}

func TestRun_MissingChecksum(t *testing.T) {
	srv := newReleaseServer(t, serverOpts{latest: "1.18.3", missingSHA: true})
	exe := newExe(t)
	_, err := newUpdater(srv.URL+"/releases", "1.18.2", exe).Run(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum file not published for Dapr CLI version 1.18.3")
	assert.Equal(t, "old", readFile(t, exe))
}

func TestRun_PackageManager(t *testing.T) {
	exe := newExe(t)
	u := newUpdater(closedServerURL(), "1.18.2", exe)
	u.DetectInstall = func(string) InstallMethod { return InstallHomebrew }
	res, err := u.Run(context.Background(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, res.PackageHint)
	assert.Equal(t, "old", readFile(t, exe))
}

func TestRun_Offline(t *testing.T) {
	exe := newExe(t)
	_, err := newUpdater(closedServerURL(), "1.18.2", exe).Run(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check for the latest Dapr CLI version")
	assert.Equal(t, "old", readFile(t, exe))
}

func TestRun_FeedNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	exe := newExe(t)
	_, err := newUpdater(srv.URL+"/releases", "1.18.2", exe).Run(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check for the latest Dapr CLI version")
}
