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

package runexec

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestWaitForApps(t *testing.T) {
	appError := errors.New("app failed")
	for _, tc := range []struct {
		name    string
		results []error
		failed  bool
	}{
		{name: "one success", results: []error{nil}},
		{name: "one failure", results: []error{appError}, failed: true},
		{name: "all succeed", results: []error{nil, nil}},
		{name: "first fails", results: []error{appError, nil}, failed: true},
		{name: "last fails", results: []error{nil, appError}, failed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			signals := make(chan os.Signal, 1)
			appExited := make(chan error)
			result := make(chan [2]bool, 1)
			t.Cleanup(func() { signals <- os.Interrupt })
			go func() {
				allExited, failed := WaitForApps(signals, appExited, len(tc.results))
				result <- [2]bool{allExited, failed}
			}()
			for i, err := range tc.results {
				select {
				case appExited <- err:
				case <-time.After(5 * time.Second):
					t.Fatal("monitor stopped accepting app exits")
				}
				if i < len(tc.results)-1 {
					assertMonitorWaiting(t, result)
				}
			}
			assertMonitorResult(t, result, [2]bool{true, tc.failed})
		})
	}
}

func TestWaitForAppsBufferedExits(t *testing.T) {
	// Both apps finish while the startup loop is still running.
	appExited := make(chan error, 2)
	appExited <- nil
	appExited <- nil
	allExited, failed := WaitForApps(make(chan os.Signal), appExited, 2)
	if !allExited || failed {
		t.Fatalf("got (%v, %v), want (true, false)", allExited, failed)
	}
}

func TestWaitForAppsSignal(t *testing.T) {
	for _, tc := range []struct {
		name     string
		appCount int
		results  []error
		failed   bool
	}{
		{name: "signal before app exits", appCount: 2},
		{name: "signal after one app exits", appCount: 2, results: []error{nil}},
		{name: "sidecar only", appCount: 1},
		{name: "mixed commands and sidecars", appCount: 3, results: []error{nil, nil}},
		{name: "empty app list"},
		{name: "failure before signal", appCount: 2, results: []error{errors.New("app failed")}, failed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			signals := make(chan os.Signal, 1)
			appExited := make(chan error, len(tc.results))
			for _, err := range tc.results {
				appExited <- err
			}
			result := make(chan [2]bool, 1)
			go func() {
				allExited, failed := WaitForApps(signals, appExited, tc.appCount)
				result <- [2]bool{allExited, failed}
			}()
			assertMonitorWaiting(t, result)
			signals <- os.Interrupt
			assertMonitorResult(t, result, [2]bool{false, tc.failed})
		})
	}
}

func TestWaitForAppsSignalPreservesQueuedFailure(t *testing.T) {
	signals := make(chan os.Signal, 1)
	signals <- os.Interrupt
	appExited := make(chan error, 1)
	appExited <- errors.New("app failed")
	allExited, failed := WaitForApps(signals, appExited, 2)
	if allExited || !failed {
		t.Fatalf("got (%v, %v), want (false, true)", allExited, failed)
	}
}

func assertMonitorWaiting(t *testing.T, result <-chan [2]bool) {
	t.Helper()
	select {
	case got := <-result:
		t.Fatalf("monitor exited before all apps finished or a signal arrived: %v", got)
	case <-time.After(25 * time.Millisecond):
	}
}

func assertMonitorResult(t *testing.T, result <-chan [2]bool, want [2]bool) {
	t.Helper()
	select {
	case got := <-result:
		if got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("monitor did not stop")
	}
}
