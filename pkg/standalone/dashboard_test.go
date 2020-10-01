// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDashboardRun(t *testing.T) {
	t.Run("build Cmd", func(t *testing.T) {
		cmd := NewDashboardCmd(9090)

		assert.Contains(t, cmd.Args[0], "dashboard")
		assert.Equal(t, cmd.Args[1], "--port")
		assert.Equal(t, cmd.Args[2], "9090")
	})
}
