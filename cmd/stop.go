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

package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
)

var stopAppID string

var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Dapr instances and their associated apps. Supported platforms: Self-hosted",
	Example: `
# Stop Dapr application
dapr stop --app-id <ID>
`,
	Run: func(cmd *cobra.Command, args []string) {
		if stopAppID != "" {
			args = append(args, stopAppID)
		}
		for _, appID := range args {
			err := standalone.Stop(appID)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "failed to stop app id %s: %s", appID, err)
			} else {
				print.SuccessStatusEvent(os.Stdout, "app stopped successfully: %s", appID)
			}
		}
	},
}

func init() {
	StopCmd.Flags().StringVarP(&stopAppID, "app-id", "a", "", "The application id to be stopped")
	StopCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(StopCmd)
}
