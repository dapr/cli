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

package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/x/dapr_test.tar.gz" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	got, err := File(tmp, srv.URL+"/x/dapr_test.tar.gz")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmp, "dapr_test.tar.gz"), got)
	content, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))
}

func TestFile_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := File(t.TempDir(), srv.URL+"/x/missing.tar.gz")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version not found")
}

func TestArchiveExt(t *testing.T) {
	if runtime.GOOS == "windows" {
		assert.Equal(t, "zip", ArchiveExt())
	} else {
		assert.Equal(t, "tar.gz", ArchiveExt())
	}
}

// writeArchive creates an archive in dir with a single entry and returns its path.
func writeArchive(t *testing.T, dir, entry, content string) string {
	t.Helper()
	path := filepath.Join(dir, "archive."+ArchiveExt())
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	if ArchiveExt() == "zip" {
		zw := zip.NewWriter(f)
		w, cerr := zw.Create(entry)
		require.NoError(t, cerr)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
		require.NoError(t, zw.Close())
		return path
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: entry, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}))
	_, err = tw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return path
}

func TestExtractBinary(t *testing.T) {
	name := "dapr"
	if runtime.GOOS == "windows" {
		name = "dapr.exe"
	}
	src := t.TempDir()
	dest := t.TempDir()
	archive := writeArchive(t, src, name, "bin")

	got, err := ExtractBinary(archive, dest, "dapr")
	require.NoError(t, err)
	content, err := os.ReadFile(got)
	require.NoError(t, err)
	assert.Equal(t, "bin", string(content))
}

func TestExtractBinary_PathTraversal(t *testing.T) {
	archive := writeArchive(t, t.TempDir(), "../evil", "x")

	_, err := ExtractBinary(archive, t.TempDir(), "dapr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "illegal file path")
}
