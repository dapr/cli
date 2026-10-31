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
	"context"
	"errors"
	"io"

	"github.com/dapr/cli/pkg/print"
)

// Runner runs an update; it is satisfied by *Updater.
type Runner interface {
	Run(ctx context.Context, requestedVersion string) (Result, error)
}

// Report runs the update with u and prints the outcome for the dapr update
// command, returning the process exit code.
func Report(ctx context.Context, u Runner, requestedVersion, goos string, stdout, stderr io.Writer) int {
	res, err := u.Run(ctx, requestedVersion)
	if err != nil {
		print.FailureStatusEvent(stderr, "Failed to update the Dapr CLI: %s", err)
		if errors.Is(err, ErrNotWritable) {
			if goos == "windows" {
				print.InfoStatusEvent(stdout, "Run the command again from an elevated (Administrator) terminal.")
			} else {
				sudoCmd := "sudo dapr update"
				if requestedVersion != "" {
					sudoCmd += " --version " + requestedVersion
				}
				print.InfoStatusEvent(stdout, "Run the command again with sudo: %s", sudoCmd)
			}
		}
		return 1
	}

	if res.PackageHint != "" {
		print.InfoStatusEvent(stdout, "%s", res.PackageHint)
		return 0
	}

	switch res.Action {
	case ActionUpToDate:
		print.InfoStatusEvent(stdout, "Dapr CLI is already on version %s.", res.From)
	case ActionAheadOfLatest:
		print.InfoStatusEvent(stdout, "Dapr CLI version %s is newer than the latest stable version %s. Use --version to install a specific version.", res.From, res.To)
	case ActionInstall:
		print.SuccessStatusEvent(stdout, "Dapr CLI updated from %s to %s.", res.From, res.To)
	}
	return 0
}
