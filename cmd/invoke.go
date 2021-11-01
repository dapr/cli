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
	"runtime"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
)

const defaultHTTPVerb = http.MethodPost

var (
	invokeAppID     string
	invokeAppMethod string
	invokeData      string
	invokeVerb      string
	invokeDataFile  string
	invokeSocket    string
)

var InvokeCmd = &cobra.Command{
	Use:   "invoke",
	Short: "Invoke a method on a given Dapr application. Supported platforms: Self-hosted",
	Example: `
# Invoke a sample method on target app with POST Verb
dapr invoke --app-id target --method sample --data '{"key":"value"}

# Invoke a sample method on target app with GET Verb
dapr invoke --app-id target --method sample --verb GET

# Invoke a sample method on target app with GET Verb using Unix domain socket
dapr invoke --unix-domain-socket --app-id target --method sample --verb GET
`,
	Run: func(cmd *cobra.Command, args []string) {
		bytePayload := []byte{}
		var err error
		if invokeDataFile != "" && invokeData != "" {
			print.FailureStatusEvent(os.Stderr, "Only one of --data and --data-file allowed in the same invoke command")
			os.Exit(1)
		}

		if invokeDataFile != "" {
			bytePayload, err = ioutil.ReadFile(invokeDataFile)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "Error reading payload from '%s'. Error: %s", invokeDataFile, err)
				os.Exit(1)
			}
		} else if invokeData != "" {
			bytePayload = []byte(invokeData)
		}
		client := standalone.NewClient()

		// TODO(@daixiang0): add Windows support
		if runtime.GOOS == "windows" && invokeSocket != "" {
			print.FailureStatusEvent(os.Stderr, "unix-domain-socket option still does not support Windows!")
			os.Exit(1)
		}

		response, err := client.Invoke(invokeAppID, invokeAppMethod, bytePayload, invokeVerb, invokeSocket)
		if err != nil {
			err = fmt.Errorf("error invoking app %s: %s", invokeAppID, err)
			print.FailureStatusEvent(os.Stderr, err.Error())
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
	InvokeCmd.Flags().StringVarP(&invokeSocket, "unix-domain-socket", "u", "", "Path to a unix domain socket dir. If specified, Dapr API servers will use Unix Domain Sockets")
	InvokeCmd.MarkFlagRequired("app-id")
	InvokeCmd.MarkFlagRequired("method")
	RootCmd.AddCommand(InvokeCmd)
}
