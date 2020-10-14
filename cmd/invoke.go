// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

const defaultHTTPVerb = http.MethodPost

var (
	invokeAppID     string
	invokeAppMethod string
	invokePayload   string
	invokeVerb      string
)

var InvokeCmd = &cobra.Command{
	Use:   "invoke",
	Short: "Invokes a Dapr app with an optional payload (deprecated, use invokePost)",
	Run: func(cmd *cobra.Command, args []string) {
		client := standalone.NewClient()
		response, err := client.Invoke(invokeAppID, invokeAppMethod, invokePayload, invokeVerb)
		if err != nil {
			err := fmt.Errorf("error invoking app %s: %s", invokeAppID, err)
			print.FailureStatusEvent(os.Stdout, err.Error())
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
	InvokeCmd.Flags().StringVarP(&invokeVerb, "verb", "v", defaultHTTPVerb, "(optional) The HTTP verb to use. default is POST")
	InvokeCmd.MarkFlagRequired("app-id")
	InvokeCmd.MarkFlagRequired("method")
	RootCmd.AddCommand(InvokeCmd)
}
