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
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), mode))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

func TestReplaceBinary_Unix(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "dapr")
	staged := cur + ".new"
	writeFile(t, cur, "old", 0o755)
	writeFile(t, staged, "new", 0o755)

	require.NoError(t, ReplaceBinary(cur, staged, "linux"))
	assert.Equal(t, "new", readFile(t, cur))
	assert.NoFileExists(t, staged)
	assert.NoFileExists(t, cur+".old")
}

func TestReplaceBinary_Windows(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "dapr.exe")
	staged := cur + ".new"
	writeFile(t, cur, "old", 0o755)
	writeFile(t, staged, "new", 0o755)

	require.NoError(t, ReplaceBinary(cur, staged, "windows"))
	assert.Equal(t, "new", readFile(t, cur))
	assert.Equal(t, "old", readFile(t, cur+".old"))
	assert.NoFileExists(t, staged)
}

func TestReplaceBinary_WindowsRollbackFailureKeepsStaged(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "dapr.exe")
	staged := cur + ".new"
	writeFile(t, cur, "old", 0o755)
	writeFile(t, staged, "new", 0o755)

	orig := rename
	t.Cleanup(func() { rename = orig })
	calls := 0
	rename = func(from, to string) error {
		calls++
		if calls >= 2 {
			return errors.New("injected failure")
		}
		return orig(from, to)
	}

	err := ReplaceBinary(cur, staged, "windows")
	require.Error(t, err)
	assert.Contains(t, err.Error(), cur+".old")
	assert.Contains(t, err.Error(), staged)
	assert.Equal(t, "new", readFile(t, staged))
	assert.Equal(t, "old", readFile(t, cur+".old"))
}

func TestReplaceBinary_WindowsOldBinaryInUse(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "dapr.exe")
	staged := cur + ".new"
	writeFile(t, cur, "old", 0o755)
	writeFile(t, staged, "new", 0o755)
	// A non-empty directory cannot be removed by os.Remove, simulating a locked .old.
	require.NoError(t, os.Mkdir(cur+".old", 0o755))
	writeFile(t, filepath.Join(cur+".old", "x"), "x", 0o644)

	err := ReplaceBinary(cur, staged, "windows")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOldBinaryInUse))
	assert.False(t, errors.Is(err, ErrNotWritable))
	assert.Equal(t, "old", readFile(t, cur))
	assert.NoFileExists(t, staged)
}

func TestReplaceBinary_WindowsRollback(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "dapr.exe")
	writeFile(t, cur, "old", 0o755)

	err := ReplaceBinary(cur, cur+".new", "windows")
	require.Error(t, err)
	assert.Equal(t, "old", readFile(t, cur))
	assert.NoFileExists(t, cur+".old")
}

func TestReplaceBinary_UnixFailureKeepsCurrent(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "dapr")
	writeFile(t, cur, "old", 0o755)

	require.Error(t, ReplaceBinary(cur, cur+".new", "linux"))
	assert.Equal(t, "old", readFile(t, cur))
}

func TestStageBinary_CopiesContentAndKeepsMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not meaningful on windows")
	}
	dir := t.TempDir()
	cur := filepath.Join(dir, "dapr")
	writeFile(t, cur, "old", 0o755)
	src := filepath.Join(t.TempDir(), "extracted")
	writeFile(t, src, "new", 0o600)
	// stale staged file must be overwritten
	writeFile(t, cur+".new", "stale-longer-content", 0o644)

	staged, err := StageBinary(cur, src)
	require.NoError(t, err)
	assert.Equal(t, cur+".new", staged)
	assert.Equal(t, "new", readFile(t, staged))
	fi, err := os.Stat(staged)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), fi.Mode().Perm())
	assert.Equal(t, "old", readFile(t, cur))
}

func TestStageBinary_EmptyExtractedPath(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "dapr")
	writeFile(t, cur, "old", 0o755)

	_, err := StageBinary(cur, "")
	require.Error(t, err)
	assert.Equal(t, "old", readFile(t, cur))
}

func TestStageBinary_MissingSourceLeavesNoStaged(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "dapr")
	writeFile(t, cur, "old", 0o755)

	_, err := StageBinary(cur, filepath.Join(dir, "nope"))
	require.Error(t, err)
	assert.NoFileExists(t, cur+".new")
}

func TestStageBinary_NotWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions are not enforced the same way on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	cur := filepath.Join(dir, "dapr")
	writeFile(t, cur, "old", 0o755)
	src := filepath.Join(t.TempDir(), "extracted")
	writeFile(t, src, "new", 0o755)
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err := StageBinary(cur, src)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotWritable))
	assert.Equal(t, "old", readFile(t, cur))
}

func TestCleanupOld(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "dapr")
	writeFile(t, cur+".old", "old", 0o755)

	CleanupOld(cur)
	assert.NoFileExists(t, cur+".old")
	CleanupOld(cur) // absent: no panic or error
}
