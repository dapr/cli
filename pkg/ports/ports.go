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

// Package ports finds out whether host ports can be bound and, if not, why
// and by whom. The CLI uses it to decide how to free the scheduler's host
// ports on Windows.
package ports

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
)

// Status is the result of trying to bind a host port.
type Status int

const (
	Free Status = iota
	// Reserved means the bind was refused although nothing listens on the
	// port. On Windows this is WSAEACCES, returned for ports inside an
	// excluded port range reserved by Hyper-V/WinNAT
	// (`netsh int ipv4 show excludedportrange protocol=tcp`). Docker reports
	// it as "ports are not available", which reads like the port is in use.
	Reserved
	// InUse means a socket is listening on the port.
	InUse
	// Unavailable is any other bind failure.
	Unavailable
)

func (s Status) String() string {
	switch s {
	case Free:
		return "free"
	case Reserved:
		return "reserved"
	case InUse:
		return "in use"
	default:
		return "unavailable"
	}
}

// Holder is who holds a port that is in use.
type Holder int

const (
	HolderUnknown Holder = iota
	// HolderWSL is WSL: its localhost relay forwarding a listener from inside
	// a distro, or the VM itself with mirrored networking. `wsl --shutdown`
	// releases it.
	HolderWSL
	// HolderContainerRuntime is the container runtime's port proxy, either
	// for a running container or for one that was just removed and whose
	// port has not been released yet.
	HolderContainerRuntime
	// HolderOther is any other process. Shutting down WSL does not help.
	HolderOther
)

func (h Holder) String() string {
	switch h {
	case HolderWSL:
		return "wsl"
	case HolderContainerRuntime:
		return "container-runtime"
	case HolderOther:
		return "other"
	default:
		return "unknown"
	}
}

// Process image names, lowercased, of WSL and container runtime processes
// that hold ports on the host.
var (
	wslProcessImages = []string{
		"wslrelay.exe", "wslhost.exe", "wslservice.exe",
		// The WSL2 VM owns the sockets with mirrored networking.
		"vmmem", "vmmem.exe", "vmmemwsl", "vmmemwsl.exe",
	}

	containerRuntimeProcessImages = []string{
		"com.docker.backend.exe", "com.docker.proxy.exe", "vpnkit.exe", "dockerd.exe",
		"docker-proxy", "dockerd", "gvproxy.exe", "gvproxy", "podman.exe", "rootlessport",
	}
)

// ClassifyHolder maps a process image name to a Holder.
func ClassifyHolder(image string) Holder {
	image = strings.ToLower(strings.TrimSpace(image))
	switch {
	case image == "":
		return HolderUnknown
	case slices.Contains(wslProcessImages, image):
		return HolderWSL
	case slices.Contains(containerRuntimeProcessImages, image):
		return HolderContainerRuntime
	default:
		return HolderOther
	}
}

// Conflict describes a host port that cannot be bound.
type Conflict struct {
	Port   int
	Status Status
	Err    error
	Holder Holder
	// PID and Process identify the listener when Status is InUse and the
	// owner could be looked up.
	PID     int
	Process string
}

func (c Conflict) String() string {
	switch {
	case c.Status == Reserved:
		return fmt.Sprintf("port %d is reserved by Windows (excluded port range) although nothing is listening on it", c.Port)
	case c.Status == InUse && c.Process != "":
		return fmt.Sprintf("port %d is in use by %s (pid %d)", c.Port, c.Process, c.PID)
	case c.Status == InUse && c.PID != 0:
		return fmt.Sprintf("port %d is in use by pid %d", c.Port, c.PID)
	case c.Status == InUse:
		return fmt.Sprintf("port %d is in use", c.Port)
	default:
		return fmt.Sprintf("port %d is not available: %v", c.Port, c.Err)
	}
}

// OwnerFunc returns the pid and process image name listening on port. It
// returns pid 0 when the owner cannot be determined. NewOwnerLookup returns
// one that resolves every port of a scan from a single system query.
type OwnerFunc func(port int) (pid int, image string)

// Scan returns a Conflict for every port that cannot be bound, in the order
// given. Each port is bound on the wildcard address and on both loopback
// addresses: WSL's localhost relay binds 127.0.0.1 or [::1] only, depending
// on the address family of the listener it forwards, and on Windows a
// wildcard bind succeeds alongside it, which would make a held port look
// free.
func Scan(owner OwnerFunc, ports ...int) []Conflict {
	var conflicts []Conflict
	for _, port := range ports {
		status, err := Probe(port)
		if status == Free {
			continue
		}
		c := Conflict{Port: port, Status: status, Err: err}
		if status == InUse && owner != nil {
			c.PID, c.Process = owner(port)
			c.Holder = ClassifyHolder(c.Process)
		}
		conflicts = append(conflicts, c)
	}
	return conflicts
}

// Probe binds port on the wildcard address, on 127.0.0.1 and on [::1]. A
// listener on any of them makes the port InUse; otherwise the first failure
// on the wildcard address or 127.0.0.1 wins. Any other failure on [::1] is
// ignored, as it also fails to bind on hosts without IPv6.
func Probe(port int) (Status, error) {
	var (
		worst    = Free
		worstErr error
	)
	for _, host := range []string{"", "127.0.0.1"} {
		status, err := Bind(host, port)
		if status == Free {
			continue
		}
		if status == InUse {
			return status, err
		}
		if worst == Free {
			worst, worstErr = status, err
		}
	}
	if status, err := Bind("::1", port); status == InUse {
		return status, err
	}
	return worst, worstErr
}

// Bind tries to listen on host:port ("" for the wildcard address) and
// classifies the result.
func Bind(host string, port int) (Status, error) {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return ClassifyBindError(err), err
	}
	ln.Close()
	return Free, nil
}

// ClassifyBindError maps a net.Listen error to a Status.
func ClassifyBindError(err error) Status {
	switch {
	case err == nil:
		return Free
	case errors.Is(err, errAddrInUse):
		return InUse
	case errors.Is(err, errAddrReserved):
		return Reserved
	default:
		return Unavailable
	}
}

// parseNetstatListenerPID returns the pid of the socket bound to port in the
// output of `netstat -ano` (Windows), or 0 if there is none. Listening (and
// bound) sockets are the rows whose foreign address is unset; the State
// column is not used because netstat translates it ("LISTENING" is
// "ABHÖREN" on German Windows). IPv4 and IPv6 both count; UDP rows have no
// state and are skipped.
//
//	Proto  Local Address          Foreign Address        State           PID
//	TCP    127.0.0.1:2379         0.0.0.0:0              LISTENING       4242
//	TCP    [::]:2379              [::]:0                 LISTENING       4242
func parseNetstatListenerPID(out string, port int) int {
	suffix := ":" + strconv.Itoa(port)
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 5 || !strings.EqualFold(fields[0], "TCP") {
			continue
		}
		if fields[2] != "0.0.0.0:0" && fields[2] != "[::]:0" {
			continue
		}
		if !strings.HasSuffix(fields[1], suffix) {
			continue
		}
		if pid, err := strconv.Atoi(fields[4]); err == nil && pid > 0 {
			return pid
		}
	}
	return 0
}

// parseTasklistImage returns the image name of pid from the output of
// `tasklist /FI "PID eq <pid>" /FO CSV /NH`, or "" if there is none. The
// record must carry the pid in its second column: when nothing matches,
// tasklist prints a translated message that may itself contain commas.
//
//	"wslrelay.exe","4242","Console","1","8,120 K"
func parseTasklistImage(out string, pid int) string {
	want := strconv.Itoa(pid)
	r := csv.NewReader(strings.NewReader(strings.TrimSpace(out)))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return ""
	}
	for _, rec := range records {
		if len(rec) >= 2 && strings.TrimSpace(rec[1]) == want {
			return strings.TrimSpace(rec[0])
		}
	}
	return ""
}

// parseExcludedRanges parses the output of
// `netsh int ipv4 show excludedportrange protocol=tcp` into [start, end]
// pairs. Ranges are printed as "     start       end" (an asterisk marks
// administered exclusions); every other line is ignored.
func parseExcludedRanges(out string) [][2]int {
	var ranges [][2]int
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		start, err1 := strconv.Atoi(fields[0])
		end, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil || start > end {
			continue
		}
		ranges = append(ranges, [2]int{start, end})
	}
	return ranges
}
