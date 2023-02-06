/*
Copyright 2023 The Dapr Authors
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
	"os"
	path_filepath "path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDaprPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "error getting home dir")

	t.Run("without flag value or env var", func(t *testing.T) {
		p, err := GetDaprRuntimePath("")
		require.NoError(t, err)
		assert.Equal(t, p, path_filepath.Join(homeDir, DefaultDaprDirName), "path should be $HOME/.dapr")
	})

	t.Run("check trim spaces", func(t *testing.T) {
		p, err := GetDaprRuntimePath("      ")
		require.NoError(t, err)
		assert.Equal(t, path_filepath.Join(homeDir, DefaultDaprDirName), p, "path should be $HOME/.dapr")

		t.Setenv("DAPR_RUNTIME_PATH", "      ")
		p, err = GetDaprRuntimePath("")
		require.NoError(t, err)
		assert.Equal(t, path_filepath.Join(homeDir, DefaultDaprDirName), p, "path should be $HOME/.dapr")
	})

	t.Run("with flag value", func(t *testing.T) {
		input := path_filepath.Join("path", "to", "dapr")
		p, err := GetDaprRuntimePath(input)
		require.NoError(t, err)
		assert.Equal(t, path_filepath.Join(input, ".dapr"), p, "path should be /path/to/dapr/.dapr")
	})

	t.Run("with env var", func(t *testing.T) {
		input := path_filepath.Join("path", "to", "dapr")
		t.Setenv("DAPR_RUNTIME_PATH", input)
		p, err := GetDaprRuntimePath("")
		require.NoError(t, err)
		assert.Equal(t, path_filepath.Join(input, ".dapr"), p, "path should be /path/to/dapr/.dapr")
	})

	t.Run("with flag value and env var", func(t *testing.T) {
		input := path_filepath.Join("path", "to", "dapr")
		input2 := path_filepath.Join("path", "to", "dapr2")
		t.Setenv("DAPR_RUNTIME_PATH", input2)
		p, err := GetDaprRuntimePath(input)
		require.NoError(t, err)
		assert.Equal(t, path_filepath.Join(input, ".dapr"), p, "path should be /path/to/dapr/.dapr")
	})
}
