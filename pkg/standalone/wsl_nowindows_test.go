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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNoopStubs verifies that every non-Windows stub returns the correct
// zero/no-op value and does not panic. This guards against accidental breakage
// of the cross-platform build contract.
func TestNoopStubs(t *testing.T) {
	assert.False(t, isWindowsElevated(), "isWindowsElevated must always be false on non-Windows")
	assert.False(t, isWSLAvailable(), "isWSLAvailable must always be false on non-Windows")
	assert.NoError(t, shutdownWSL())
	assert.NoError(t, stopWinNAT())
	assert.NoError(t, startWinNAT())
	startWSLBackground() // must not panic
}
