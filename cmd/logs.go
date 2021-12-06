// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
)

var (
	logsAppID string
	podName   string
	namespace string
	k8s       bool
)

var LogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Get Dapr sidecar logs for an application. Supported platforms: Kubernetes",
	Example: `
# Get logs of sample app from target pod in custom namespace
dapr logs -k --app-id sample --pod-name target --namespace custom
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := kubernetes.Logs(logsAppID, podName, namespace)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}
		print.SuccessStatusEvent(os.Stdout, "Fetched logs")
	},
}

func init() {
	LogsCmd.Flags().BoolVarP(&k8s, "kubernetes", "k", true, "Get logs from a Kubernetes cluster")
	LogsCmd.Flags().StringVarP(&logsAppID, "app-id", "a", "", "The application id for which logs are needed")
	LogsCmd.Flags().StringVarP(&podName, "pod-name", "p", "", "The name of the pod in Kubernetes, in case your application has multiple pods (optional)")
	LogsCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "The Kubernetes namespace in which your application is deployed")
	LogsCmd.Flags().BoolP("help", "h", false, "Print this help message")
	LogsCmd.MarkFlagRequired("app-id")
	LogsCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(LogsCmd)
}
