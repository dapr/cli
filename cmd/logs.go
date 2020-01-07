// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"os"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/spf13/cobra"
)

var logsAppID string
var logsFor string
var namespace string

var LogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Gets logs for a Dapr app in Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		err := kubernetes.Logs(logsAppID, logsFor, namespace)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}
		print.SuccessStatusEvent(os.Stdout, "Fetched logs")
	},
}

func init() {
	LogsCmd.Flags().StringVarP(&logsAppID, "app-id", "a", "", "the app id for which logs are needed")
	LogsCmd.Flags().StringVarP(&logsFor, "for", "f", "", "logs for? possible values: dapr or app")
	LogsCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "kubernetes namespace in which your application is deployed. default value is 'default'")
	LogsCmd.MarkFlagRequired("app-id")
	LogsCmd.MarkFlagRequired("for")
	RootCmd.AddCommand(LogsCmd)
}
