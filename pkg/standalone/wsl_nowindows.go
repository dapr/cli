//go:build !windows

/*
Copyright 2021 The Dapr Authors
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

package standalone

// isWindowsElevated always returns false on non-Windows platforms.
func isWindowsElevated() bool { return false }

// isWSLAvailable always returns false on non-Windows platforms.
func isWSLAvailable() bool { return false }

// shutdownWSL is a no-op on non-Windows platforms.
func shutdownWSL() error { return nil }

// stopWinNAT is a no-op on non-Windows platforms.
func stopWinNAT() error { return nil }

// startWinNAT is a no-op on non-Windows platforms.
func startWinNAT() error { return nil }

// startWSLBackground is a no-op on non-Windows platforms.
func startWSLBackground() {}
