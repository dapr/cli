/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
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
	Short: "Invoke a method on a given Dapr application. Supported platforms: Kubernetes and self-hosted",
	Example: `
# Invoke a sample method on target app with POST Verb in self-hosted mode
dapr invoke --app-id target --method sample --data '{"key":"value"}'

# Invoke a sample method on target app with in Kubernetes
dapr invoke -k --app-id target --method sample --data '{"key":"value"}'


# Invoke a sample method on target app with GET Verb in self-hosted mode
dapr invoke --app-id target --method sample --verb GET

# Invoke a sample method on target app with GET Verb using Unix domain socket in self-hosted mode
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

		// TODO(@daixiang0): add Windows support
		if invokeSocket != "" {
			if runtime.GOOS == "windows" {
				print.FailureStatusEvent(os.Stderr, "The unix-domain-socket option is not supported on Windows")
				os.Exit(1)
			} else {
				print.WarningStatusEvent(os.Stdout, "Unix domain sockets are currently a preview feature")
			}
		}

		var response string
		if kubernetesMode {
			response, err = kubernetes.Invoke(invokeAppID, invokeAppMethod, bytePayload, invokeVerb)
		} else {
			client := standalone.NewClient()
			response, err = client.Invoke(invokeAppID, invokeAppMethod, bytePayload, invokeVerb, invokeSocket)
		}
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
	InvokeCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Invoke a method on a Dapr application in a Kubernetes cluster")
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
