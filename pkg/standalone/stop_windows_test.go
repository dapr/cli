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

package standalone

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/kolesnikovae/go-winjob"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"

	"github.com/dapr/cli/utils"
)

func TestDisposeJobHandleRetainsStopEvent(t *testing.T) {
	// Only the helper is assigned to the job, never the test process.
	helper := exec.Command(os.Args[0], "-test.run=^TestStopEventHelperProcess$")
	helper.Env = append(os.Environ(), "DAPR_CLI_STOP_EVENT_HELPER=1")
	require.NoError(t, helper.Start())
	wait := sync.OnceValue(func() error { return helper.Wait() })
	t.Cleanup(func() {
		_ = helper.Process.Kill()
		_ = wait()
	})
	helperExited := make(chan error, 1)
	go func() { helperExited <- wait() }()

	cliPID := helper.Process.Pid
	eventName, err := windows.UTF16PtrFromString(fmt.Sprintf("dapr_cli_%v", cliPID))
	require.NoError(t, err)
	eventOwner, err := windows.CreateEvent(nil, 0, 0, eventName)
	require.NoError(t, err)
	closeOwner := sync.OnceValue(func() error { return windows.CloseHandle(eventOwner) })
	t.Cleanup(func() { _ = closeOwner() })

	job, err := winjob.Create(utils.GetJobObjectNameFromPID(strconv.Itoa(cliPID)))
	require.NoError(t, err)
	t.Cleanup(func() { _ = job.Close() })
	require.NoError(t, job.Assign(helper.Process))

	// Use the existing post-termination delay as a synchronization barrier.
	// Before disposal continues, the helper must have exited and the original
	// event handle must be closed. The old open-after-wait ordering therefore
	// cannot pass just because an observer goroutine happened to run late.
	// Do not retain another event handle in this test: that would hide the race.
	waitCalled := false
	err = disposeJobHandleWithWait(cliPID, func(delay time.Duration) {
		waitCalled = true
		require.Equal(t, 5*time.Second, delay)
		select {
		case helperErr := <-helperExited:
			require.NoError(t, helperErr, "job termination should return the requested zero exit status")
		case <-time.After(5 * time.Second):
			t.Fatal("job termination did not stop the helper")
		}
		require.NoError(t, closeOwner(), "the original event handle must close before disposal continues")
	})
	require.NoError(t, err)
	require.True(t, waitCalled, "the test must exercise the post-termination barrier")
}

func TestSetStopEvent(t *testing.T) {
	cliPID := os.Getpid()
	eventName, err := windows.UTF16PtrFromString(fmt.Sprintf("dapr_cli_%v", cliPID))
	require.NoError(t, err)
	event, err := windows.CreateEvent(nil, 0, 0, eventName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = windows.CloseHandle(event) })

	require.NoError(t, setStopEvent(cliPID))
	status, err := windows.WaitForSingleObject(event, 0)
	require.NoError(t, err)
	require.EqualValues(t, windows.WAIT_OBJECT_0, status)
}

func TestStopEventHelperProcess(t *testing.T) {
	if os.Getenv("DAPR_CLI_STOP_EVENT_HELPER") != "1" {
		t.Skip("helper process for TestDisposeJobHandleRetainsStopEvent")
	}
	// Parent cleanup normally terminates this process. Bound an orphaned helper.
	time.Sleep(30 * time.Second)
	os.Exit(0)
}
