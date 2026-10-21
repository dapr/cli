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
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		if uninstallEntryExists(root) {
			return true
		}
	}
	return false
}

func uninstallEntryExists(root registry.Key) bool {
	parent, err := registry.OpenKey(root, uninstallKeyPath, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return false
	}
	defer parent.Close()
	names, err := parent.ReadSubKeyNames(-1)
	if err != nil {
		return false
	}
	for _, name := range names {
		sub, err := registry.OpenKey(root, uninstallKeyPath+`\`+name, registry.QUERY_VALUE)
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
