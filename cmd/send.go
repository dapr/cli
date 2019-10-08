// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/send"
	"github.com/spf13/cobra"
)

var sendAppID string
var sendAppMethod string
var sendPayload string

var SendCmd = &cobra.Command{
	Use:   "send",
	Short: "invoke a dapr app with an optional payload",
	Run: func(cmd *cobra.Command, args []string) {
		response, err := send.InvokeApp(sendAppID, sendAppMethod, sendPayload)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error invoking app %s: %s", sendAppID, err))
			return
		}

		if response != "" {
			fmt.Println(response)
		}

		print.SuccessStatusEvent(os.Stdout, "App invoked successfully")
	},
}

func init() {
	SendCmd.Flags().StringVarP(&sendAppID, "app-id", "a", "", "the app id to invoke")
	SendCmd.Flags().StringVarP(&sendAppMethod, "method", "m", "", "the method to invoke")
	SendCmd.Flags().StringVarP(&sendPayload, "payload", "p", "", "(optional) a json payload")
	SendCmd.MarkFlagRequired("app-id")
	SendCmd.MarkFlagRequired("method")
	RootCmd.AddCommand(SendCmd)
}
