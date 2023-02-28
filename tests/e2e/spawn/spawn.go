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

package spawn

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// CommandWithContext runs a command with its arguments in background.
// The provided context is used to kill the command (by calling os.Process.Kill)
// if the context becomes done before the command completes on its own.
// The return channels can be used to read stdout & stderr.
func CommandWithContext(ctx context.Context, command string, arguments ...string) (chan string, chan string, error) {
	cmd := exec.CommandContext(ctx, command, arguments...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, nil, err
	}

	stdOutChan := make(chan string)
	stdErrChan := make(chan string)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			stdOutChan <- scanner.Text()
		}
		close(stdOutChan)
		cmd.Wait()
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stdErrChan <- scanner.Text()
		}
		close(stdErrChan)
		cmd.Wait()
	}()

	return stdOutChan, stdErrChan, nil
}

// Command runs a command with its arguments and returns the stdout or stderr or the error.
func Command(command string, arguments ...string) (string, error) {
	cmd := exec.Command(command, arguments...)

	outBytes, err := cmd.CombinedOutput()
	if err != nil && outBytes == nil {
		return "", err
	}

	return string(outBytes), err
}

// CommandExecWithContext runs a command with its arguments, kills the command after context is done
// and returns the combined stdout, stderr or the error.
func CommandExecWithContext(ctx context.Context, command string, arguments ...string) (string, error) {
	cmd := exec.Command(command, arguments...)
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("error starting command : %w", err)
	}

	waitErrChan := make(chan error, 1)
	go func(errChan chan error) {
		waitErr := cmd.Wait()
		if waitErr != nil {
			fmt.Printf("error waiting for command : %s\n", waitErr)
		}
		waitErrChan <- waitErr
	}(waitErrChan)
	<-ctx.Done()
	if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
		cmd.Process.Signal(syscall.SIGTERM)
		time.Sleep(10 * time.Second)
		if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
			err = cmd.Process.Kill()
			if err != nil {
				return b.String(), fmt.Errorf("error killing command : %w", err)
			}
		}
	}

	return b.String(), <-waitErrChan
}
