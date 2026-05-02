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
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckPorts exercises the core port-availability helper used by the
// Windows WSL2 port-conflict detection path.
func TestCheckPorts(t *testing.T) {
	t.Run("returns nil when given no ports", func(t *testing.T) {
		assert.NoError(t, checkPorts())
	})

	t.Run("returns nil when all ports are free", func(t *testing.T) {
		p1 := freePort(t)
		p2 := freePort(t)
		assert.NoError(t, checkPorts(p1, p2))
	})

	t.Run("returns error containing port number when port is in use", func(t *testing.T) {
		ln := holdPort(t)
		defer ln.Close()
		port := ln.Addr().(*net.TCPAddr).Port

		err := checkPorts(port)
		require.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("port %d", port))
	})

	t.Run("returns error for first occupied port in the list", func(t *testing.T) {
		free := freePort(t)

		ln := holdPort(t)
		defer ln.Close()
		busy := ln.Addr().(*net.TCPAddr).Port

		// free comes first — we still expect failure once the busy port is reached.
		err := checkPorts(free, busy)
		require.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("port %d", busy))
	})

	t.Run("succeeds for free port preceding an occupied one when first fails fast", func(t *testing.T) {
		ln := holdPort(t)
		defer ln.Close()
		busy := ln.Addr().(*net.TCPAddr).Port

		free := freePort(t)

		// busy comes first — error must name it, not the free port.
		err := checkPorts(busy, free)
		require.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("port %d", busy))
		assert.NotContains(t, err.Error(), fmt.Sprintf("port %d", free))
	})
}

// TestCheckSchedulerPorts_PortInUse verifies that checkSchedulerPorts surfaces
// an error (with the port number) when the gRPC port it is given is already
// bound. This is the scenario triggered by WSL2 holding scheduler ports.
func TestCheckSchedulerPorts_PortInUse(t *testing.T) {
	ln := holdPort(t)
	defer ln.Close()
	busyPort := ln.Addr().(*net.TCPAddr).Port

	err := checkSchedulerPorts(busyPort)
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("port %d", busyPort))
}

// holdPort binds an OS-assigned port and returns the listener. The caller is
// responsible for closing it. Using ":0" matches the binding style of
// utils.CheckIfPortAvailable so the conflict is detected reliably.
func holdPort(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	return ln
}

// freePort returns an available port number by briefly binding then releasing
// it. There is a theoretical race between Close and the subsequent
// checkPorts call, but in practice this window is negligible for unit tests.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}
