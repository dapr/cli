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
	invokeAppID     string
	invokeAppMethod string
	invokePayload   string
)

var InvokeCmd = &cobra.Command{
	Use:   "invoke",
	Short: "Invokes a Dapr app with an optional payload (deprecated, use invokePost)",
	Run: func(cmd *cobra.Command, args []string) {
		response, err := invoke.Post(invokeAppID, invokeAppMethod, invokePayload)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error invoking app %s: %s", invokeAppID, err))
			return
		}

		if response != "" {
			fmt.Println(response)
		}

		print.SuccessStatusEvent(os.Stdout, "App invoked successfully")
	},
}

func init() {
	InvokeCmd.Flags().StringVarP(&invokeAppID, "app-id", "a", "", "the app id to invoke")
	InvokeCmd.Flags().StringVarP(&invokeAppMethod, "method", "m", "", "the method to invoke")
	InvokeCmd.Flags().StringVarP(&invokePayload, "payload", "p", "", "(optional) a json payload")
	InvokeCmd.MarkFlagRequired("app-id")
	InvokeCmd.MarkFlagRequired("method")
	RootCmd.AddCommand(InvokeCmd)
}
