//go:build e2e && !template

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

package standalone_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapr/cli/tests/e2e/common"
	"github.com/dapr/cli/tests/e2e/spawn"
)

// updateTargetVersion is a published release the update tests install.
const updateTargetVersion = "1.18.2"

// copyDaprBinary copies the e2e dapr binary into dir so update tests never
// replace the binary used by the other tests.
func copyDaprBinary(t *testing.T, dir string) (string, []byte) {
	t.Helper()
	src := common.GetDaprPath()
	if runtime.GOOS == "windows" {
		src += ".exe"
	}
	content, err := os.ReadFile(src)
	require.NoError(t, err)
	dst := filepath.Join(dir, filepath.Base(src))
	require.NoError(t, os.WriteFile(dst, content, 0o755)) //nolint:gosec
	return dst, content
}

func assertBinaryUnchanged(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(want, got), "binary %s was modified", path)
}

func TestStandaloneUpdate(t *testing.T) {
	t.Run("invalid version", func(t *testing.T) {
		dapr, content := copyDaprBinary(t, t.TempDir())
		output, err := spawn.Command(dapr, "update", "--version", "latest")
		t.Log(output)
		require.Error(t, err, "expected update with an invalid version to fail")
		assert.Contains(t, output, "invalid version")
		assertBinaryUnchanged(t, dapr, content)
	})

	t.Run("read-only install directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("directory permissions work differently on Windows")
		}
		if os.Geteuid() == 0 {
			t.Skip("root can write to read-only directories")
		}
		dir := t.TempDir()
		dapr, content := copyDaprBinary(t, dir)
		require.NoError(t, os.Chmod(dir, 0o555))       //nolint:gosec
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) //nolint:gosec

		output, err := spawn.Command(dapr, "update", "--version", updateTargetVersion)
		t.Log(output)
		require.Error(t, err, "expected update in a read-only directory to fail")
		assert.Contains(t, output, "sudo dapr update --version "+updateTargetVersion)
		assertBinaryUnchanged(t, dapr, content)
	})

	t.Run("update to a specific version", func(t *testing.T) {
		dapr, _ := copyDaprBinary(t, t.TempDir())
		output, err := spawn.Command(dapr, "update", "--version", updateTargetVersion)
		t.Log(output)
		require.NoError(t, err, "dapr update failed")
		assert.Contains(t, output, "updated from ")
		assert.Contains(t, output, " to "+updateTargetVersion)

		output, err = spawn.Command(dapr, "version")
		t.Log(output)
		require.NoError(t, err, "dapr version failed")
		assert.Contains(t, output, "CLI version: "+updateTargetVersion)

		// Running "update --version" again on the installed binary to check for
		// "already on version" needs a target release that ships "dapr update"
		// itself; 1.18.2 predates it.
	})
}
