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
	"os"
	"runtime"

	"github.com/spf13/cobra"

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
		if code := selfupdate.Report(context.Background(), u, updateVersion, runtime.GOOS, os.Stdout, os.Stderr); code != 0 {
			os.Exit(code)
		}
	},
}

func init() {
	UpdateCmd.Flags().StringVar(&updateVersion, "version", "", "The version of the Dapr CLI to install (default: latest stable version)")
	UpdateCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(UpdateCmd)
}
