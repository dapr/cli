package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

var (
	invokeAppID     string
	invokeAppMethod string
	invokePayload   string
)

// invokeGetCmd represents the invokeGet command.
var invokeGetCmd = &cobra.Command{
	Use:   "invokeGet",
	Short: "Issue HTTP GET to Dapr app",
	Run: func(cmd *cobra.Command, args []string) {
		client := standalone.NewClient()
		response, err := client.InvokeGet(invokeAppID, invokeAppMethod)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("error invoking app %s: %s", invokeAppID, err))
			// exit with error
			os.Exit(1)
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
