// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"io/ioutil"
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
	invokeData      string
	invokeVerb      string
	invokeDataFile  string
)

var InvokeCmd = &cobra.Command{
	Use:   "invoke",
	Short: "Invoke a method on a given Dapr application. Supported platforms: Self-hosted",
	Example: `
# Invoke a sample method on target app with POST Verb
dapr invoke --app-id target --method sample --data '{"key":"value"}

# Invoke a sample method on target app with GET Verb
dapr invoke --app-id target --method sample --verb GET
`,
	Run: func(cmd *cobra.Command, args []string) {
		bytePayload := []byte{}
		var err error
		if invokeDataFile != "" && invokeData != "" {
			print.FailureStatusEvent(os.Stdout, "Only one of --data and --data-file allowed in the same invoke command")
			os.Exit(1)
		}

		if invokeDataFile != "" {
			bytePayload, err = ioutil.ReadFile(invokeDataFile)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, "Error reading payload from '%s'. Error: %s", invokeDataFile, err)
				os.Exit(1)
			}
		} else if invokeData != "" {
			bytePayload = []byte(invokeData)
		}
		client := standalone.NewClient()
		response, err := client.Invoke(invokeAppID, invokeAppMethod, bytePayload, invokeVerb)
		if err != nil {
			err = fmt.Errorf("error invoking app %s: %s", invokeAppID, err)
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
	InvokeCmd.Flags().StringVarP(&invokeAppID, "app-id", "a", "", "The application id to invoke")
	InvokeCmd.Flags().StringVarP(&invokeAppMethod, "method", "m", "", "The method to invoke")
	InvokeCmd.Flags().StringVarP(&invokeData, "data", "d", "", "The JSON serialized data string (optional)")
	InvokeCmd.Flags().StringVarP(&invokeVerb, "verb", "v", defaultHTTPVerb, "The HTTP verb to use")
	InvokeCmd.Flags().StringVarP(&invokeDataFile, "data-file", "f", "", "A file containing the JSON serialized data (optional)")
	InvokeCmd.Flags().BoolP("help", "h", false, "Print this help message")
	InvokeCmd.MarkFlagRequired("app-id")
	InvokeCmd.MarkFlagRequired("method")
	RootCmd.AddCommand(InvokeCmd)
}
