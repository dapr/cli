/*
Copyright 2026 The Dapr Authors
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

package ports

import (
	"errors"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScan(t *testing.T) {
	noOwner := func(int) (int, string) { return 0, "" }

	t.Run("returns nothing when given no ports", func(t *testing.T) {
		assert.Empty(t, Scan(noOwner))
	})

	t.Run("returns nothing when all ports are free", func(t *testing.T) {
		// Port 0 always binds (the OS selects a free ephemeral port), so there
		// is no bind/close race here.
		assert.Empty(t, Scan(noOwner, 0, 0))
	})

	t.Run("reports a port held on the wildcard address as in use", func(t *testing.T) {
		port := holdPort(t, "")

		conflicts := Scan(noOwner, port)
		require.Len(t, conflicts, 1)
		assert.Equal(t, port, conflicts[0].Port)
		assert.Equal(t, InUse, conflicts[0].Status)
		assert.Contains(t, conflicts[0].String(), fmt.Sprintf("port %d is in use", port))
	})

	// WSL's localhost relay (wslrelay.exe) binds 127.0.0.1 only. On Windows a
	// wildcard bind succeeds next to it, so a wildcard-only check reported
	// the port as free while Docker's bind then failed.
	t.Run("reports a port held on loopback only as in use", func(t *testing.T) {
		port := holdPort(t, "127.0.0.1")

		conflicts := Scan(noOwner, port)
		require.Len(t, conflicts, 1)
		assert.Equal(t, InUse, conflicts[0].Status)
	})

	t.Run("reports every busy port in order, skipping free ones", func(t *testing.T) {
		first, second := holdPort(t, ""), holdPort(t, "")

		conflicts := Scan(noOwner, second, 0, first)
		require.Len(t, conflicts, 2)
		assert.Equal(t, second, conflicts[0].Port)
		assert.Equal(t, first, conflicts[1].Port)
	})

	t.Run("looks up and classifies the owner of a port in use", func(t *testing.T) {
		port := holdPort(t, "")
		var looked []int
		owner := func(p int) (int, string) { looked = append(looked, p); return 4242, "wslrelay.exe" }

		conflicts := Scan(owner, port)
		require.Len(t, conflicts, 1)
		assert.Equal(t, []int{port}, looked)
		assert.Equal(t, HolderWSL, conflicts[0].Holder)
		assert.Equal(t, 4242, conflicts[0].PID)
		assert.Equal(t, fmt.Sprintf("port %d is in use by wslrelay.exe (pid 4242)", port), conflicts[0].String())
	})

	t.Run("leaves the holder unknown when the owner cannot be found", func(t *testing.T) {
		port := holdPort(t, "")

		conflicts := Scan(noOwner, port)
		require.Len(t, conflicts, 1)
		assert.Equal(t, HolderUnknown, conflicts[0].Holder)
	})

	t.Run("names the pid when the image is unknown", func(t *testing.T) {
		port := holdPort(t, "")
		owner := func(int) (int, string) { return 4242, "" }

		conflicts := Scan(owner, port)
		require.Len(t, conflicts, 1)
		assert.Equal(t, HolderUnknown, conflicts[0].Holder)
		assert.Equal(t, fmt.Sprintf("port %d is in use by pid 4242", port), conflicts[0].String())
	})
}

// TestScanDetectsHeldPorts checks the scanner against real sockets: a
// listener on the wildcard address and one on either loopback address only
// must all be reported as in use, and a free port as free.
func TestScanDetectsHeldPorts(t *testing.T) {
	hosts := []string{"", "127.0.0.1"}
	if ln, err := net.Listen("tcp", "[::1]:0"); err == nil {
		ln.Close()
		hosts = append(hosts, "::1")
	} else {
		t.Logf("no IPv6 loopback on this host, not testing it: %v", err)
	}
	for _, host := range hosts {
		ln, port := listen(t, host)
		conflicts := Scan(NewOwnerLookup(), port)
		ln.Close()
		require.Len(t, conflicts, 1, "listener on %q:%d", host, port)
		assert.Equal(t, InUse, conflicts[0].Status, "listener on %q:%d: %v", host, port, conflicts[0])
	}
	assert.Empty(t, Scan(NewOwnerLookup(), 0), "port 0 always binds")
}

// TestClassifyBindError checks that a reserved port (nobody listening) is
// told apart from a port in use. Docker reports both as "ports are not
// available", which is how a reserved port got mistaken for one held by WSL.
func TestClassifyBindError(t *testing.T) {
	bindErr := func(errno error) error {
		return &net.OpError{Op: "listen", Net: "tcp", Err: os.NewSyscallError("bind", errno)}
	}

	assert.Equal(t, Free, ClassifyBindError(nil))
	assert.Equal(t, InUse, ClassifyBindError(bindErr(errAddrInUse)))
	assert.Equal(t, Reserved, ClassifyBindError(bindErr(errAddrReserved)))
	assert.Equal(t, Unavailable, ClassifyBindError(errors.New("something else")))

	assert.Contains(t, Conflict{Port: 2379, Status: Reserved}.String(), "reserved by Windows")
	assert.Contains(t, Conflict{Port: 2379, Status: Unavailable, Err: errors.New("boom")}.String(), "boom")
}

func TestClassifyHolder(t *testing.T) {
	for image, want := range map[string]Holder{
		"":                       HolderUnknown,
		"wslrelay.exe":           HolderWSL,
		"WSLRelay.exe":           HolderWSL,
		"wslhost.exe":            HolderWSL,
		"vmmemWSL":               HolderWSL,
		"vmmem.exe":              HolderWSL,
		"com.docker.backend.exe": HolderContainerRuntime,
		"vpnkit.exe":             HolderContainerRuntime,
		"gvproxy.exe":            HolderContainerRuntime,
		"docker-proxy":           HolderContainerRuntime,
		"etcd.exe":               HolderOther,
		"svchost.exe":            HolderOther,
	} {
		assert.Equal(t, want, ClassifyHolder(image), image)
	}
}

func TestParseNetstatListenerPID(t *testing.T) {
	const out = `
Active Connections

  Proto  Local Address          Foreign Address        State           PID
  TCP    0.0.0.0:135            0.0.0.0:0              LISTENING       1100
  TCP    0.0.0.0:12379          0.0.0.0:0              LISTENING       7000
  TCP    127.0.0.1:50000        127.0.0.1:2379         ESTABLISHED     7100
  TCP    127.0.0.1:2379         127.0.0.1:50001        TIME_WAIT       0
  TCP    127.0.0.1:2379         0.0.0.0:0              LISTENING       4242
  TCP    [::]:58081             [::]:0                 LISTENING       5151
  TCP    [::1]:59091            [::]:0                 LISTENING       6161
  UDP    0.0.0.0:6060           *:*                                    7200
  UDP    [::]:6060              *:*                                    7200
`
	assert.Equal(t, 4242, parseNetstatListenerPID(out, 2379), "only the listening socket on the local port counts")
	assert.Equal(t, 5151, parseNetstatListenerPID(out, 58081), "IPv6 listeners count")
	assert.Equal(t, 6161, parseNetstatListenerPID(out, 59091), "IPv6 loopback listeners count")
	assert.Equal(t, 7000, parseNetstatListenerPID(out, 12379))
	assert.Equal(t, 0, parseNetstatListenerPID(out, 379), "a port must not match the tail of a longer one")
	assert.Equal(t, 0, parseNetstatListenerPID(out, 6060), "UDP sockets are not listeners")
	assert.Equal(t, 0, parseNetstatListenerPID(out, 50006))
	assert.Equal(t, 0, parseNetstatListenerPID("", 2379))

	// netstat translates the State column; the match must not depend on it.
	const german = `
  Proto  Lokale Adresse         Remoteadresse          Status          PID
  TCP    0.0.0.0:2379           0.0.0.0:0              ABHÖREN         4242
  TCP    127.0.0.1:50000        127.0.0.1:2379         HERGESTELLT     7100
`
	assert.Equal(t, 4242, parseNetstatListenerPID(german, 2379))
}

func TestParseTasklistImage(t *testing.T) {
	assert.Equal(t, "wslrelay.exe", parseTasklistImage(`"wslrelay.exe","4242","Console","1","8,120 K"`+"\r\n", 4242))
	assert.Equal(t, "com.docker.backend.exe", parseTasklistImage(`"com.docker.backend.exe","88","Console","1","61,004 K"`, 88))
	assert.Empty(t, parseTasklistImage(`"wslrelay.exe","4242","Console","1","8,120 K"`, 4243), "a record for another pid")
	assert.Empty(t, parseTasklistImage("INFO: No tasks are running which match the specified criteria.\r\n", 4242))
	assert.Empty(t, parseTasklistImage("INFORMATION: Es werden keine Tasks ausgeführt, die den angegebenen Kriterien entsprechen.\r\n", 4242),
		"a translated no-match message with commas is not an image")
	assert.Empty(t, parseTasklistImage("", 4242))
}

func TestParseExcludedRanges(t *testing.T) {
	const out = `

Protocol tcp Port Exclusion Ranges

Start Port    End Port
----------    --------
      1024        1123
      2379        2379     *
     50000       50059

* - Administered port exclusions.

`
	assert.Equal(t, [][2]int{{1024, 1123}, {2379, 2379}, {50000, 50059}}, parseExcludedRanges(out))
	assert.Empty(t, parseExcludedRanges(""))
}

// holdPort binds an OS-assigned port on host ("" for the wildcard address)
// until the test ends and returns the port.
func holdPort(t *testing.T, host string) int {
	t.Helper()
	ln, port := listen(t, host)
	t.Cleanup(func() { ln.Close() })
	return port
}

// listen binds an OS-assigned port on host ("" for the wildcard address).
func listen(t *testing.T, host string) (net.Listener, int) {
	t.Helper()
	ln, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	require.NoError(t, err)
	return ln, ln.Addr().(*net.TCPAddr).Port
}
