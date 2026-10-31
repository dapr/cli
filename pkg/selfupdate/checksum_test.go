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
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeChecksumFixture(t *testing.T, format func(sum string) string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	content := []byte("hello dapr")
	filePath := filepath.Join(dir, "dapr_linux_amd64.tar.gz")
	require.NoError(t, os.WriteFile(filePath, content, 0o600))
	sum := sha256.Sum256(content)
	checksumPath := filepath.Join(dir, "dapr_linux_amd64.tar.gz.sha256")
	require.NoError(t, os.WriteFile(checksumPath, []byte(format(hex.EncodeToString(sum[:]))), 0o600))
	return filePath, checksumPath
}

func TestVerifyChecksum_OK(t *testing.T) {
	f, c := writeChecksumFixture(t, func(s string) string { return s + " *dapr_linux_amd64.tar.gz\n" })
	assert.NoError(t, VerifyChecksum(f, c))
}

func TestVerifyChecksum_Uppercase(t *testing.T) {
	f, c := writeChecksumFixture(t, func(s string) string { return strings.ToUpper(s) + " *dapr_linux_amd64.tar.gz\n" })
	assert.NoError(t, VerifyChecksum(f, c))
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	f, c := writeChecksumFixture(t, func(string) string {
		return strings.Repeat("0", 64) + " *dapr_linux_amd64.tar.gz\n"
	})
	err := VerifyChecksum(f, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestVerifyChecksum_Empty(t *testing.T) {
	f, c := writeChecksumFixture(t, func(string) string { return "" })
	assert.Error(t, VerifyChecksum(f, c))
}
