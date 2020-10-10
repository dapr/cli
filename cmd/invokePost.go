// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

var invokePostCmd = &cobra.Command{
	Use:   "invokePost",
	Short: "Issue HTTP POST to Dapr app with an optional payload",
	Run: func(cmd *cobra.Command, args []string) {
		err := invokePost(invokeAppID, invokeAppMethod, invokePayload)
		if err != nil {
			// exit with error
			os.Exit(1)
		}
		print.SuccessStatusEvent(os.Stdout, fmt.Sprintf("HTTP Post to method %s invoked successfully", invokeAppMethod))
	},
}

func invokePost(invokeAppID, invokeAppMethod, invokePayload string) error {
	client := standalone.NewClient()
	response, err := client.Post(invokeAppID, invokeAppMethod, invokePayload)
	if err != nil {
		er := fmt.Errorf("error invoking app %s: %s", invokeAppID, err)
		print.FailureStatusEvent(os.Stdout, er.Error())
		return er
	}

	if response != "" {
		fmt.Println(response)
	}
	return nil
}

func init() {
	invokePostCmd.Flags().StringVarP(&invokeAppID, "app-id", "a", "", "the app id to invoke")
	invokePostCmd.Flags().StringVarP(&invokeAppMethod, "method", "m", "", "the method to invoke")
	invokePostCmd.Flags().StringVarP(&invokePayload, "payload", "p", "", "(optional) a json payload")

	invokePostCmd.MarkFlagRequired("app-id")
	invokePostCmd.MarkFlagRequired("method")

	RootCmd.AddCommand(invokePostCmd)
}
