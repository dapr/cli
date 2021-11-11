// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"os"

	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the health status of Dapr services. Supported platforms: Kubernetes",
	Example: `
# Get status of Dapr services from Kubernetes
dapr status -k 
`,
	Run: func(cmd *cobra.Command, args []string) {
		sc, err := kubernetes.NewStatusClient()
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}
		status, err := sc.Status()
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}
		if len(status) == 0 {
			print.FailureStatusEvent(os.Stderr, "No status returned. Is Dapr initialized in your cluster?")
			os.Exit(1)
		}
		table, err := gocsv.MarshalString(status)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}

		utils.PrintTable(table)
	},
}

func init() {
	StatusCmd.Flags().BoolVarP(&k8s, "kubernetes", "k", false, "Show the health status of Dapr services on Kubernetes cluster")
	StatusCmd.Flags().BoolP("help", "h", false, "Print this help message")
	StatusCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(StatusCmd)
}
