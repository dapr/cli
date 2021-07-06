// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListRunWhenNoDaprrd(t *testing.T) {
	t.Run("test Cmd", func(t *testing.T) {
		startDashboard := exec.Command("dapr", "dashboard", "-p", "5555")
		startDashboard.Run()
		cmd := exec.Command("Dapr", "list")
		output := cmd.Run()
		assert.Empty(t, output)
	})
}
