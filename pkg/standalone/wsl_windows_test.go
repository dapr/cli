//go:build windows

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

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsWindowsElevated_Callable verifies the function completes without
// panicking. The actual return value depends on whether the test process is
// running as Administrator, so we only log it rather than assert a fixed value.
func TestIsWindowsElevated_Callable(t *testing.T) {
	elevated := isWindowsElevated()
	t.Logf("isWindowsElevated() = %v (test process running as Administrator: %v)", elevated, elevated)
}

// TestIsWSLAvailable_MatchesLookPath verifies that isWSLAvailable reports the
// same result as exec.LookPath("wsl"), confirming it accurately reflects
// whether wsl.exe is on the PATH.
func TestIsWSLAvailable_MatchesLookPath(t *testing.T) {
	_, err := exec.LookPath("wsl")
	expected := err == nil
	assert.Equal(t, expected, isWSLAvailable(),
		"isWSLAvailable() should return true iff wsl.exe is on PATH")
}

// TestStartWSLBackground_DoesNotBlock verifies that startWSLBackground returns
// promptly regardless of whether WSL is installed. When WSL is absent the
// internal cmd.Start() fails silently; when present, wsl --exec echo exits
// immediately and the cleanup goroutine reaps it.
func TestStartWSLBackground_DoesNotBlock(t *testing.T) {
	// This must complete without hanging; no assertion on side-effects.
	startWSLBackground()
}
