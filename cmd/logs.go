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
var podName string
var namespace string
var k8s bool

var LogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Gets Dapr sidecar logs for an app in Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		err := kubernetes.Logs(logsAppID, podName, namespace)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}
		print.SuccessStatusEvent(os.Stdout, "Fetched logs")
	},
}

func init() {
	LogsCmd.Flags().BoolVarP(&k8s, "kubernetes", "k", true, "Only works with a Kubernetes cluster")
	LogsCmd.Flags().StringVarP(&logsAppID, "app-id", "a", "", "The app id for which logs are needed")
	LogsCmd.Flags().StringVarP(&podName, "pod-name", "p", "", "(optional) Name of the Pod. Use this in case you have multiple app instances (Pods)")
	LogsCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "(optional) Kubernetes namespace in which your application is deployed. default value is 'default'")
	LogsCmd.MarkFlagRequired("app-id")
	LogsCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(LogsCmd)
}
