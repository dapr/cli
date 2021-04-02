// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
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
				print.FailureStatusEvent(os.Stdout, "failed to stop app id %s: %s", appID, err)
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
