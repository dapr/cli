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

func TestDashboardRun(t *testing.T) {
	t.Parallel()
	t.Run("build Cmd", func(t *testing.T) {
		cmd, err := NewDashboardCmd("", 9090)

		assert.NoError(t, err)
		assert.Contains(t, cmd.Args[0], "dashboard")
		assert.Equal(t, cmd.Args[1], "--port")
		assert.Equal(t, cmd.Args[2], "9090")
	})

	t.Run("start dashboard on random free port", func(t *testing.T) {
		cmd, err := NewDashboardCmd("", 0)

		assert.NoError(t, err)
		assert.Contains(t, cmd.Args[0], "dashboard")
		assert.Equal(t, cmd.Args[1], "--port")
		assert.NotEqual(t, cmd.Args[2], "0")
	})
}
