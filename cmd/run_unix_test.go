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

package cmd

import (
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKillProcessGroup_KillsEntireGroup verifies that killProcessGroup terminates all
// processes in the group, not just the leader. A second process is explicitly placed
// into the leader's process group using SysProcAttr.Pgid; both must be gone after
// killProcessGroup returns.
func TestKillProcessGroup_KillsEntireGroup(t *testing.T) {
	leader := exec.Command("sleep", "100")
	leader.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, leader.Start())

	// Explicitly place a second process in the same process group as the leader.
	member := exec.Command("sleep", "100")
	member.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: leader.Process.Pid}
	require.NoError(t, member.Start())

	// Wait goroutines reap the processes once killed, so Kill(-pgid, 0) eventually
	// returns ESRCH rather than seeing zombies that still count as group members.
	go func() { _ = leader.Wait() }()
	go func() { _ = member.Wait() }()

	require.NoError(t, killProcessGroup(leader.Process))

	assert.Eventually(t, func() bool {
		return syscall.Kill(-leader.Process.Pid, 0) == syscall.ESRCH
	}, 2*time.Second, 50*time.Millisecond, "process group should be gone after killProcessGroup")
}

// TestKillProcessGroup_AlreadyGone verifies that killProcessGroup returns nil when
// the process has already exited cleanly before kill is attempted.
func TestKillProcessGroup_AlreadyGone(t *testing.T) {
	cmd := exec.Command("true")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, cmd.Start())
	proc := cmd.Process
	require.NoError(t, cmd.Wait())

	assert.NoError(t, killProcessGroup(proc))
}

// TestKillProcessGroup_LeaderExitedGroupAlive verifies that killProcessGroup still
// kills remaining group members when the leader has already exited — the typical
// `go run` scenario where the wrapper process exits but the compiled binary lives on
// in the same process group.
func TestKillProcessGroup_LeaderExitedGroupAlive(t *testing.T) {
	leader := exec.Command("sleep", "100")
	leader.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, leader.Start())
	leaderPGID := leader.Process.Pid

	// Place a second process in the same group as the leader.
	member := exec.Command("sleep", "100")
	member.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: leaderPGID}
	require.NoError(t, member.Start())

	// Kill only the group leader; member stays alive in the same group.
	require.NoError(t, leader.Process.Kill())
	_ = leader.Wait() // reap immediately so it doesn't linger as a zombie

	// Confirm the group still has a live member before we call killProcessGroup.
	require.NoError(t, syscall.Kill(-leaderPGID, 0), "member should still be alive in the group")

	// killProcessGroup falls back to process.Pid as PGID since Getpgid returns ESRCH
	// for the dead leader, then signals and terminates the remaining member.
	go func() { _ = member.Wait() }()

	require.NoError(t, killProcessGroup(leader.Process))

	assert.Eventually(t, func() bool {
		return syscall.Kill(-leaderPGID, 0) == syscall.ESRCH
	}, 2*time.Second, 50*time.Millisecond, "process group should be gone after leader exits")
}
