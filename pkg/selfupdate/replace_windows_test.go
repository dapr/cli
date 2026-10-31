//go:build windows

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
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const helperEnv = "DAPR_SELFUPDATE_HELPER"

// TestHelperProcess is not a real test: it keeps a copy of the test binary
// running so tests can replace a running executable. The parent test kills it;
// the long sleep only bounds an orphaned helper.
func TestHelperProcess(t *testing.T) {
	if os.Getenv(helperEnv) != "1" {
		t.Skip("helper process for TestReplaceBinary_WindowsRunningBinary")
	}
	time.Sleep(10 * time.Minute)
	os.Exit(0)
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	require.NoError(t, err)
	defer in.Close()
	out, err := os.Create(dst)
	require.NoError(t, err)
	_, err = io.Copy(out, in)
	require.NoError(t, err)
	require.NoError(t, out.Close())
}

func TestReplaceBinary_WindowsRunningBinary(t *testing.T) {
	testExe, err := os.Executable()
	require.NoError(t, err)
	current := filepath.Join(t.TempDir(), "dapr.exe")
	copyFile(t, testExe, current)

	cmd := exec.Command(current, "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), helperEnv+"=1")
	require.NoError(t, cmd.Start())
	stop := sync.OnceFunc(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	// Registered after t.TempDir, so it runs before the temp dir is removed.
	t.Cleanup(stop)

	// A running image can't be opened for writing.
	require.Eventually(t, func() bool {
		f, openErr := os.OpenFile(current, os.O_WRONLY, 0)
		if openErr != nil {
			return true
		}
		f.Close()
		return false
	}, 10*time.Second, 50*time.Millisecond, "helper process did not start")

	staged := current + ".new"
	writeFile(t, staged, "new", 0o755)
	require.NoError(t, ReplaceBinary(current, staged, "windows"))
	assert.Equal(t, "new", readFile(t, current))
	assert.FileExists(t, current+".old")

	// The running image (now .old) can't be deleted.
	CleanupOld(current)
	assert.FileExists(t, current+".old")

	writeFile(t, staged, "newer", 0o755)
	err = ReplaceBinary(current, staged, "windows")
	require.ErrorIs(t, err, ErrOldBinaryInUse)
	assert.Equal(t, "new", readFile(t, current))

	stop()
	assert.Eventually(t, func() bool {
		CleanupOld(current)
		_, statErr := os.Lstat(current + ".old")
		return errors.Is(statErr, fs.ErrNotExist)
	}, 10*time.Second, 100*time.Millisecond, ".old not removed after the process exited")
}
