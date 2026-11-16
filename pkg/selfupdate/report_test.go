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

package selfupdate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeRunner struct {
	res       Result
	err       error
	requested string
}

func (f *fakeRunner) Run(_ context.Context, requestedVersion string) (Result, error) {
	f.requested = requestedVersion
	return f.res, f.err
}

func TestReport(t *testing.T) {
	notWritable := fmt.Errorf("%w: /usr/local/bin: permission denied", ErrNotWritable)
	oldInUse := fmt.Errorf("%w: C:\\dapr\\dapr.exe.old", ErrOldBinaryInUse)

	tests := []struct {
		name       string
		res        Result
		err        error
		requested  string
		goos       string
		wantCode   int
		wantStdout []string
		wantStderr []string
		notOutput  []string
	}{
		{
			name:       "package hint",
			res:        Result{From: "1.18.2", PackageHint: "Installed with Homebrew. Run:\n  brew upgrade dapr/tap/dapr-cli"},
			goos:       "darwin",
			wantStdout: []string{"Installed with Homebrew. Run:\n  brew upgrade dapr/tap/dapr-cli"},
			notOutput:  []string{"updated"},
		},
		{
			name:       "package hint with percent sign",
			res:        Result{PackageHint: "100% managed by %s %d"},
			goos:       "darwin",
			wantStdout: []string{"100% managed by %s %d"},
			notOutput:  []string{"%!"},
		},
		{
			name:       "up to date",
			res:        Result{Action: ActionUpToDate, From: "1.18.3", To: "1.18.3"},
			goos:       "linux",
			wantStdout: []string{"Dapr CLI is already on version 1.18.3."},
		},
		{
			name:       "ahead of latest",
			res:        Result{Action: ActionAheadOfLatest, From: "1.19.0-rc.1", To: "1.18.3"},
			goos:       "linux",
			wantStdout: []string{"Dapr CLI version 1.19.0-rc.1 is newer than the latest stable version 1.18.3."},
		},
		{
			name:       "installed",
			res:        Result{Action: ActionInstall, From: "1.18.2", To: "1.18.3"},
			goos:       "linux",
			wantStdout: []string{"Dapr CLI updated from 1.18.2 to 1.18.3."},
		},
		{
			name:       "not writable on linux",
			err:        notWritable,
			goos:       "linux",
			wantCode:   1,
			wantStdout: []string{"Run the command again with sudo: sudo dapr update\n"},
			wantStderr: []string{"Failed to update the Dapr CLI: install directory is not writable"},
			notOutput:  []string{"Administrator"},
		},
		{
			name:       "not writable on linux with version",
			err:        notWritable,
			requested:  "1.17.0",
			goos:       "linux",
			wantCode:   1,
			wantStdout: []string{"sudo dapr update --version 1.17.0"},
			wantStderr: []string{"Failed to update the Dapr CLI"},
		},
		{
			name:       "not writable on windows",
			err:        notWritable,
			goos:       "windows",
			wantCode:   1,
			wantStdout: []string{"elevated (Administrator) terminal"},
			wantStderr: []string{"Failed to update the Dapr CLI"},
			notOutput:  []string{"sudo"},
		},
		{
			name:       "old binary in use",
			err:        oldInUse,
			goos:       "windows",
			wantCode:   1,
			wantStderr: []string{"Failed to update the Dapr CLI: a previous dapr binary is still in use"},
			notOutput:  []string{"sudo", "Administrator"},
		},
		{
			name:       "generic error",
			err:        errors.New("boom"),
			goos:       "linux",
			wantCode:   1,
			wantStderr: []string{"Failed to update the Dapr CLI: boom"},
			notOutput:  []string{"sudo", "Administrator"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := &fakeRunner{res: tc.res, err: tc.err}
			var stdout, stderr bytes.Buffer

			code := Report(context.Background(), u, tc.requested, tc.goos, &stdout, &stderr)

			assert.Equal(t, tc.wantCode, code)
			assert.Equal(t, tc.requested, u.requested)
			for _, s := range tc.wantStdout {
				assert.Contains(t, stdout.String(), s)
			}
			for _, s := range tc.wantStderr {
				assert.Contains(t, stderr.String(), s)
			}
			if tc.err == nil {
				assert.Empty(t, stderr.String())
			} else {
				assert.NotContains(t, stdout.String(), "updated")
			}
			for _, s := range tc.notOutput {
				assert.NotContains(t, stdout.String()+stderr.String(), s)
			}
		})
	}
}
