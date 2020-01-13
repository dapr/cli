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

// invokeGetCmd represents the invokeGet command
var invokeGetCmd = &cobra.Command{
	Use:   "invokeGet",
	Short: "Issue HTTP GET to Dapr app",
	Run: func(cmd *cobra.Command, args []string) {
		response, err := invoke.Get(invokeAppID, invokeAppMethod)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error invoking app %s: %s", invokeAppID, err))

			return
		}

		if response != "" {
			fmt.Println(response)
		}

		print.SuccessStatusEvent(os.Stdout, fmt.Sprintf("HTTP Get to method %s invoked successfully", invokeAppMethod))
	},
}

func init() {
	invokeGetCmd.Flags().StringVarP(&invokeAppID, "app-id", "a", "", "the app id to invoke")
	invokeGetCmd.Flags().StringVarP(&invokeAppMethod, "method", "m", "", "the method to invoke")

	invokeGetCmd.MarkFlagRequired("app-id")
	invokeGetCmd.MarkFlagRequired("method")

	RootCmd.AddCommand(invokeGetCmd)
}
