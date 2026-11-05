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
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	runExec "github.com/dapr/cli/pkg/runexec"
)

func TestStartAppProcessReportsExit(t *testing.T) {
	for _, tc := range []struct {
		code       string
		retainPipe bool
	}{
		{code: "0"},
		{code: "7"},
		{code: "0", retainPipe: true},
		{code: "7", retainPipe: true},
	} {
		t.Run(fmt.Sprintf("exit %s retained pipe %v", tc.code, tc.retainPipe), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			command := exec.Command(os.Args[0], "-test.run=^TestMultiAppHelperProcess$")
			command.Env = append(os.Environ(), "DAPR_CLI_MULTI_APP_HELPER="+tc.code)
			if tc.retainPipe {
				releaseFile := filepath.Join(t.TempDir(), "release")
				command.Env = append(command.Env, "DAPR_CLI_MULTI_APP_RETAIN_PIPE="+releaseFile)
				t.Cleanup(func() {
					if err := os.WriteFile(releaseFile, nil, 0o600); err != nil {
						t.Error(err)
						return
					}
					deadline := time.Now().Add(5 * time.Second)
					for time.Now().Before(deadline) {
						if _, err := os.Stat(releaseFile + ".done"); err == nil {
							return
						}
						time.Sleep(10 * time.Millisecond)
					}
					t.Error("output-holding helper did not stop")
				})
			}
			runState := &runExec.RunExec{
				AppID: "test-app",
				AppCMD: &runExec.CmdProcess{
					Command: command, OutputWriter: &stdout, ErrorWriter: &stderr,
				},
			}
			appRunning := make(chan bool, 1)
			appExited := make(chan error, 1)
			startErrors := make(chan error, 1)
			startAppProcess(runState, appRunning, appExited, startErrors)
			if !<-appRunning {
				t.Fatalf("start failed: %v", <-startErrors)
			}
			t.Cleanup(func() { _ = command.Process.Kill() })
			select {
			case err := <-appExited:
				if (err != nil) != (tc.code != "0") {
					t.Fatalf("exit code %s reported %v", tc.code, err)
				}
				if tc.code == "7" {
					var exitErr *exec.ExitError
					if !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
						t.Fatalf("original app exit status was lost: %v", err)
					}
				}
			case <-time.After(10 * time.Second):
				t.Fatal("app did not report its exit")
			}
			if tc.retainPipe && !strings.Contains(stderr.String(), "App output did not close") {
				t.Fatal("missing warning about inherited output pipes")
			}
			// A completion result must not race with the final log writes.
			for i := range 2000 {
				if !strings.Contains(stdout.String(), fmt.Sprintf("stdout line %d\n", i)) {
					t.Fatalf("stdout line %d missing after app exit", i)
				}
				if !strings.Contains(stderr.String(), fmt.Sprintf("stderr line %d\n", i)) {
					t.Fatalf("stderr line %d missing after app exit", i)
				}
			}
			select {
			case err := <-appExited:
				t.Fatalf("duplicate exit notification: %v", err)
			default:
			}
		})
	}
}

func TestStartAppProcessNoCommand(t *testing.T) {
	appRunning := make(chan bool, 1)
	appExited := make(chan error, 1)
	startErrors := make(chan error, 1)
	startAppProcess(&runExec.RunExec{AppCMD: &runExec.CmdProcess{}}, appRunning, appExited, startErrors)
	if !<-appRunning {
		t.Fatal("sidecar-only app should start without a command")
	}
	select {
	case err := <-appExited:
		t.Fatalf("sidecar-only app reported a process exit: %v", err)
	default:
	}
}

func TestStartAppProcessStartFailure(t *testing.T) {
	command := exec.Command(filepath.Join(t.TempDir(), "missing-command"))
	runState := &runExec.RunExec{
		AppCMD: &runExec.CmdProcess{Command: command, OutputWriter: io.Discard, ErrorWriter: io.Discard},
	}
	appRunning := make(chan bool, 1)
	appExited := make(chan error, 1)
	startErrors := make(chan error, 1)
	startAppProcess(runState, appRunning, appExited, startErrors)
	if <-appRunning {
		t.Fatal("missing command unexpectedly started")
	}
	if err := <-startErrors; err == nil {
		t.Fatal("missing startup error")
	}
	select {
	case err := <-appExited:
		t.Fatalf("failed startup reported a process exit: %v", err)
	default:
	}
}

func TestMultiAppHelperProcess(t *testing.T) {
	code := os.Getenv("DAPR_CLI_MULTI_APP_HELPER")
	if code == "" {
		return
	}
	releaseFile := os.Getenv("DAPR_CLI_MULTI_APP_RETAIN_PIPE")
	if code == "hold" {
		// Retain the parent's output descriptors after it exits. The deadline
		// bounds this helper even if the test fails before releasing it.
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(releaseFile); err == nil {
				_ = os.WriteFile(releaseFile+".done", nil, 0o600)
				os.Exit(0)
			}
			time.Sleep(10 * time.Millisecond)
		}
		os.Exit(0)
	}
	if releaseFile != "" {
		child := exec.Command(os.Args[0], "-test.run=^TestMultiAppHelperProcess$")
		child.Env = append(os.Environ(), "DAPR_CLI_MULTI_APP_HELPER=hold")
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	for i := range 2000 {
		fmt.Fprintf(os.Stdout, "stdout line %d\n", i)
		fmt.Fprintf(os.Stderr, "stderr line %d\n", i)
	}
	if code == "7" {
		os.Exit(7)
	}
	os.Exit(0)
}
