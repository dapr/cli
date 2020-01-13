// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/invoke"
	"github.com/dapr/cli/pkg/print"
	"github.com/spf13/cobra"
)

var invokeResourceID string

// invokeDeleteCmd represents the invokeDelete command
var invokeDeleteCmd = &cobra.Command{
	Use:   "invokeDelete",
	Short: "Issue HTTP DELETE to Dapr app",
	Run: func(cmd *cobra.Command, args []string) {
		err := invoke.Delete(invokeAppID, invokeAppMethod, invokeResourceID)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error invoking app %s: %s", invokeAppID, err))
			return
		}

		print.SuccessStatusEvent(os.Stdout, fmt.Sprintf("HTTP Delete to method %s invoked successfully", invokeAppMethod))
	},
}

func init() {
	invokeDeleteCmd.Flags().StringVarP(&invokeAppID, "app-id", "a", "", "the app id to invoke")
	invokeDeleteCmd.Flags().StringVarP(&invokeAppMethod, "method", "m", "", "the method to invoke")
	invokeDeleteCmd.Flags().StringVarP(&invokeResourceID, "id", "", "", "the resource id")

	invokeDeleteCmd.MarkFlagRequired("app-id")
	invokeDeleteCmd.MarkFlagRequired("method")
	invokeDeleteCmd.MarkFlagRequired("id")

	RootCmd.AddCommand(invokeDeleteCmd)
}
