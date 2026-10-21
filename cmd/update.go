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
		updater := &selfupdate.Updater{CurrentVersion: cliVersion}
		res, err := updater.Run(context.Background(), updateVersion)
		if err != nil {
			if errors.Is(err, selfupdate.ErrOldBinaryInUse) {
				print.FailureStatusEvent(os.Stderr, "Failed to update the Dapr CLI: %s", err)
				os.Exit(1)
			}
			print.FailureStatusEvent(os.Stderr, "Failed to update the Dapr CLI: %s", err)
			if errors.Is(err, selfupdate.ErrNotWritable) {
				if runtime.GOOS == "windows" {
					print.InfoStatusEvent(os.Stdout, "Run the command again from an elevated (Administrator) terminal.")
				} else {
					print.InfoStatusEvent(os.Stdout, "Run the command again with sudo: sudo dapr update")
				}
			}
			os.Exit(1)
		}

		if res.PackageHint != "" {
			print.InfoStatusEvent(os.Stdout, "%s", res.PackageHint)
			return
		}

		switch res.Action {
		case selfupdate.ActionUpToDate:
			print.InfoStatusEvent(os.Stdout, "Dapr CLI is already on version %s.", res.From)
		case selfupdate.ActionAheadOfLatest:
			print.InfoStatusEvent(os.Stdout, "Dapr CLI version %s is newer than the latest stable version %s. Use --version to install a specific version.", res.From, res.To)
		case selfupdate.ActionInstall:
			print.SuccessStatusEvent(os.Stdout, "Dapr CLI updated from %s to %s.", res.From, res.To)
		}
	},
}

func init() {
	UpdateCmd.Flags().StringVar(&updateVersion, "version", "", "The version of the Dapr CLI to install (default: latest stable version)")
	UpdateCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(UpdateCmd)
}
