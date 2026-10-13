//go:build windows

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
	"fmt"
	"strings"
	"sync"
	"syscall"

	"github.com/dapr/cli/utils"
)

const (
	// errAddrInUse is WSAEADDRINUSE: a socket is listening on the port.
	errAddrInUse = syscall.Errno(10048)
	// errAddrReserved is WSAEACCES: the port is in an excluded port range.
	errAddrReserved = syscall.Errno(10013)
)

// NewOwnerLookup returns an OwnerFunc backed by `netstat -ano` and
// `tasklist`. netstat is run once, on the first port that needs it, and its
// output is reused for every port of the same scan. No `-p TCP`: that lists
// IPv4 sockets only, and a listener on [::] (what Go's net.Listen(":port")
// creates) would be missed.
func NewOwnerLookup() OwnerFunc {
	netstat := sync.OnceValue(func() string {
		out, err := utils.RunCmdAndWait("netstat", "-ano")
		if err != nil {
			return ""
		}
		return out
	})
	return func(port int) (int, string) {
		pid := parseNetstatListenerPID(netstat(), port)
		if pid == 0 {
			return 0, ""
		}
		out, err := utils.RunCmdAndWait("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
		if err != nil {
			return pid, ""
		}
		return pid, parseTasklistImage(out, pid)
	}
}

// IsWinNATRunning reports whether the Windows NAT driver service (WinNat) is
// running. It reserves the dynamic excluded port ranges.
func IsWinNATRunning() bool {
	out, err := utils.RunCmdAndWait("sc", "query", "winnat")
	return err == nil && strings.Contains(out, "RUNNING")
}

// ExcludedRanges returns the TCP port ranges Windows refuses to bind
// (`netsh int ipv4 show excludedportrange protocol=tcp`).
func ExcludedRanges() ([][2]int, error) {
	out, err := utils.RunCmdAndWait("netsh", "int", "ipv4", "show", "excludedportrange", "protocol=tcp")
	if err != nil {
		return nil, err
	}
	return parseExcludedRanges(out), nil
}
