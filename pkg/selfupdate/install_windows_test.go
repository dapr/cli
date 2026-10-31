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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

const scratchKeyBase = `SOFTWARE\DaprCLISelfUpdateTest`

// useScratchUninstallKey points uninstallKeyPath at a unique, not yet created
// key under HKCU and removes it again when the test ends.
func useScratchUninstallKey(t *testing.T) string {
	t.Helper()
	path := scratchKeyBase + `\` + strconv.FormatInt(time.Now().UnixNano(), 10)
	orig := uninstallKeyPath
	uninstallKeyPath = path
	t.Cleanup(func() {
		uninstallKeyPath = orig
		deleteKeyTree(path)
		// Only succeeds when no other test still uses the base key.
		_ = registry.DeleteKey(registry.CURRENT_USER, scratchKeyBase)
	})
	return path
}

func deleteKeyTree(path string) {
	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return
	}
	names, _ := k.ReadSubKeyNames(-1)
	k.Close()
	for _, name := range names {
		_ = registry.DeleteKey(registry.CURRENT_USER, path+`\`+name)
	}
	_ = registry.DeleteKey(registry.CURRENT_USER, path)
}

func createKey(t *testing.T, path string, values map[string]string) {
	t.Helper()
	k, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
	require.NoError(t, err)
	defer k.Close()
	for name, value := range values {
		require.NoError(t, k.SetStringValue(name, value))
	}
}

func TestMSIInstalled_DaprEntry(t *testing.T) {
	path := useScratchUninstallKey(t)
	createKey(t, path+`\{other}`, map[string]string{"DisplayName": "Other App", "Publisher": "Other"})
	createKey(t, path+`\{dapr}`, map[string]string{"DisplayName": "Dapr CLI", "Publisher": "Dapr"})
	assert.True(t, msiInstalled())
}

func TestMSIInstalled_OtherPublisher(t *testing.T) {
	path := useScratchUninstallKey(t)
	createKey(t, path+`\{dapr}`, map[string]string{"DisplayName": "Dapr CLI", "Publisher": "Someone Else"})
	assert.False(t, msiInstalled())
}

func TestMSIInstalled_EmptyParentKey(t *testing.T) {
	path := useScratchUninstallKey(t)
	createKey(t, path, nil)
	assert.False(t, msiInstalled())
}

func TestMSIInstalled_MissingParentKey(t *testing.T) {
	useScratchUninstallKey(t)
	assert.False(t, msiInstalled())
}
