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
	"golang.org/x/sys/windows/registry"
)

const uninstallKeyPath = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`

// msiInstalled reports whether the Dapr CLI MSI is registered under the
// HKCU or HKLM uninstall keys.
func msiInstalled() bool {
	// The MSI is built as a 32-bit package, so per-machine installs register
	// under the WOW6432Node view of HKLM, which a 64-bit process does not see
	// by default; check that view explicitly as well.
	const (
		defaultView = 0
		wow64View   = registry.WOW64_32KEY
	)
	if uninstallEntryExists(registry.CURRENT_USER, defaultView) ||
		uninstallEntryExists(registry.LOCAL_MACHINE, defaultView) ||
		uninstallEntryExists(registry.LOCAL_MACHINE, wow64View) {
		return true
	}
	return false
}

func uninstallEntryExists(root registry.Key, view uint32) bool {
	parent, err := registry.OpenKey(root, uninstallKeyPath, registry.ENUMERATE_SUB_KEYS|view)
	if err != nil {
		return false
	}
	defer parent.Close()
	names, err := parent.ReadSubKeyNames(-1)
	if err != nil {
		return false
	}
	for _, name := range names {
		sub, err := registry.OpenKey(root, uninstallKeyPath+`\`+name, registry.QUERY_VALUE|view)
		if err != nil {
			continue
		}
		displayName, _, err1 := sub.GetStringValue("DisplayName")
		publisher, _, err2 := sub.GetStringValue("Publisher")
		sub.Close()
		if err1 == nil && err2 == nil && displayName == "Dapr CLI" && publisher == "Dapr" {
			return true
		}
	}
	return false
}
