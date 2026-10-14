//go:build !windows

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

import "syscall"

const (
	errAddrInUse = syscall.EADDRINUSE
	// errAddrReserved is EACCES, e.g. a privileged port bound without root.
	errAddrReserved = syscall.EACCES
)

// NewOwnerLookup returns an OwnerFunc that never finds an owner: port
// remedies are only needed on Windows.
func NewOwnerLookup() OwnerFunc {
	return func(int) (int, string) { return 0, "" }
}

// IsWinNATRunning is always false off Windows.
func IsWinNATRunning() bool { return false }

// ExcludedRanges is empty off Windows.
func ExcludedRanges() ([][2]int, error) { return nil, nil }
