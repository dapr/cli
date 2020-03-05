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

var (
	invokePostCmd = &cobra.Command{
		Use:   "invokePost",
		Short: "Issue HTTP POST to Dapr app with an optional payload",
		Run: func(cmd *cobra.Command, args []string) {

			response, err := invoke.Post(invokeAppID, invokeAppMethod, invokePayload)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error invoking app %s: %s", invokeAppID, err))

				return
			}

			if response != "" {
				fmt.Println(response)
			}

			print.SuccessStatusEvent(os.Stdout, fmt.Sprintf("HTTP Post to method %s invoked successfully", invokeAppMethod))

		},
	}
)

func init() {
	invokePostCmd.Flags().StringVarP(&invokeAppID, "app-id", "a", "", "the app id to invoke")
	invokePostCmd.Flags().StringVarP(&invokeAppMethod, "method", "m", "", "the method to invoke")
	invokePostCmd.Flags().StringVarP(&invokePayload, "payload", "p", "", "(optional) a json payload")

	invokePostCmd.MarkFlagRequired("app-id")
	invokePostCmd.MarkFlagRequired("method")

	RootCmd.AddCommand(invokePostCmd)
}
