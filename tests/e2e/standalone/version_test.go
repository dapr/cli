//go:build e2e
// +build e2e

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
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandaloneVersion(t *testing.T) {
	ensureDaprInstallation(t)

	t.Run("version", func(t *testing.T) {
		output, err := cmdVersion("")
		t.Log(output)
		require.NoError(t, err, "dapr version failed")
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 2, "expected at least 2 fields in components outptu")
		assert.Contains(t, lines[0], "CLI version")
		assert.Contains(t, lines[1], "Runtime version")
	})

	t.Run("version json", func(t *testing.T) {
		output, err := cmdVersion("json")
		t.Log(output)
		require.NoError(t, err, "dapr version failed")
		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		assert.NoError(t, err, "output was not valid JSON")
	})
}
