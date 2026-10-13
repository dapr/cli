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
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOwnerLookup_FindsThisProcess runs the real netstat/tasklist lookup
// against ports held by the test binary itself, on the wildcard address (a
// dual-stack [::] socket) and on IPv4 loopback.
func TestOwnerLookup_FindsThisProcess(t *testing.T) {
	for _, host := range []string{"", "127.0.0.1"} {
		ln, port := listen(t, host)
		pid, image := NewOwnerLookup()(port)
		conflicts := Scan(NewOwnerLookup(), port)
		ln.Close()

		assert.Equal(t, os.Getpid(), pid, "listener on %q:%d", host, port)
		assert.True(t, strings.EqualFold(filepath.Base(os.Args[0]), image), "image %q, test binary %q", image, os.Args[0])
		require.Len(t, conflicts, 1)
		assert.Equal(t, InUse, conflicts[0].Status)
		assert.Equal(t, HolderOther, conflicts[0].Holder, "a non-WSL, non-runtime process")
	}
}

// TestProbe_LoopbackOnlyListener checks the case of WSL's wslrelay, which
// binds 127.0.0.1 only. It also logs whether a wildcard-only check would have
// seen the conflict, which is the behavior the loopback probe guards against.
func TestProbe_LoopbackOnlyListener(t *testing.T) {
	port := holdPort(t, "127.0.0.1")

	if ln, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(port))); err == nil {
		ln.Close()
		t.Logf("wildcard bind of %d SUCCEEDED next to a loopback listener: a wildcard-only check misses it", port)
	} else {
		t.Logf("wildcard bind of %d failed next to a loopback listener: %v", port, err)
	}

	status, err := Probe(port)
	assert.Equal(t, InUse, status, "err: %v", err)
}

// TestProbe_ExcludedPortRange binds a port inside one of the host's excluded
// port ranges, if it has any, and expects it to be reported as reserved
// (WSAEACCES) rather than in use.
//
// It probes the first port of each range until one has no listener.
func TestProbe_ExcludedPortRange(t *testing.T) {
	ranges, err := ExcludedRanges()
	if err != nil {
		t.Skipf("cannot list excluded port ranges: %v", err)
	}
	for _, r := range ranges {
		status, err := Probe(r[0])
		if status == InUse {
			continue // the service that reserved the range is listening on it
		}
		assert.Equal(t, Reserved, status, "port %d in excluded range %d-%d: %v", r[0], r[0], r[1], err)
		return
	}
	t.Skip("no excluded port range without a listener on this host")
}
