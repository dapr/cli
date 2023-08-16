//go:build e2e && !template
// +build e2e,!template

/*
Copyright 2022 The Dapr Authors
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
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStandaloneInitNegatives(t *testing.T) {
	// Ensure a clean environment
	must(t, cmdUninstall, "failed to uninstall Dapr")

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "expected no error on querying for os home dir")

	t.Run("run without install", func(t *testing.T) {
		t.Parallel()
		output, err := cmdRun("")
		require.Error(t, err, "expected error status on run without install")
		path := filepath.Join(homeDir, ".dapr", "components")
		if runtime.GOOS == "windows" {
			require.Contains(t, output, path+": The system cannot find the path specified")
		} else {
			require.Contains(t, output, path+": no such file or directory", "expected output to contain message")
		}
	})

	t.Run("list without install", func(t *testing.T) {
		t.Parallel()
		output, err := cmdList("")
		require.NoError(t, err, "expected no error status on list without install")
		require.Equal(t, "No Dapr instances found.\n", output)
	})

	t.Run("stop without install", func(t *testing.T) {
		t.Parallel()
		output, err := cmdStopWithAppID("test")
		require.NoError(t, err, "expected no error on stop without install")
		require.Contains(t, output, "failed to stop app id test: couldn't find app id test", "expected output to match")
	})

	t.Run("uninstall without install", func(t *testing.T) {
		t.Parallel()
		output, err := cmdUninstall()
		require.NoError(t, err, "expected no error on uninstall without install")
		require.Contains(t, output, "Removing Dapr from your machine...", "expected output to contain message")
		path := filepath.Join(homeDir, ".dapr", "bin")
		require.Contains(t, output, "WARNING: "+path+" does not exist", "expected output to contain message")
		if !isSlimMode() {
			require.Contains(t, output, "WARNING: dapr_placement container does not exist", "expected output to contain message")
			require.Contains(t, output, "WARNING: dapr_redis container does not exist", "expected output to contain message")
			require.Contains(t, output, "WARNING: dapr_zipkin container does not exist", "expected output to contain message")
		}
		path = filepath.Join(homeDir, ".dapr")
		require.Contains(t, output, "WARNING: "+path+" does not exist", "expected output to contain message")
		require.Contains(t, output, "Dapr has been removed successfully")
	})
}
