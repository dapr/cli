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
	"context"
	"errors"
	"io"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/selfupdate"
)

var updateVersion string

// UpdateCmd is a command for updating the Dapr CLI itself.
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the Dapr CLI to the latest stable version or a specific version",
	Example: `
# Update the Dapr CLI to the latest stable version
dapr update

# Update the Dapr CLI to a specific version
dapr update --version 1.18.3
`,
	Run: func(cmd *cobra.Command, args []string) {
		u := &selfupdate.Updater{CurrentVersion: cliVersion}
		if code := runUpdate(context.Background(), u, updateVersion, runtime.GOOS, os.Stdout, os.Stderr); code != 0 {
			os.Exit(code)
		}
	},
}

// updater is the part of selfupdate.Updater used by the update command.
type updater interface {
	Run(ctx context.Context, requestedVersion string) (selfupdate.Result, error)
}

// runUpdate runs the update and reports the outcome, returning the exit code.
func runUpdate(ctx context.Context, u updater, requestedVersion, goos string, stdout, stderr io.Writer) int {
	res, err := u.Run(ctx, requestedVersion)
	if err != nil {
		print.FailureStatusEvent(stderr, "Failed to update the Dapr CLI: %s", err)
		if errors.Is(err, selfupdate.ErrNotWritable) {
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
	case selfupdate.ActionUpToDate:
		print.InfoStatusEvent(stdout, "Dapr CLI is already on version %s.", res.From)
	case selfupdate.ActionAheadOfLatest:
		print.InfoStatusEvent(stdout, "Dapr CLI version %s is newer than the latest stable version %s. Use --version to install a specific version.", res.From, res.To)
	case selfupdate.ActionInstall:
		print.SuccessStatusEvent(stdout, "Dapr CLI updated from %s to %s.", res.From, res.To)
	}
	return 0
}

func init() {
	UpdateCmd.Flags().StringVar(&updateVersion, "version", "", "The version of the Dapr CLI to install (default: latest stable version)")
	UpdateCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(UpdateCmd)
}
