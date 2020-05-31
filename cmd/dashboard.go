// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/spf13/cobra"
)

var open bool

var DashboardCmd = &cobra.Command{
	Use: "dashboard",
	Short: "Runs the Dapr dashboard on local machine or on a Kubernetes cluster",
	Run: func(cmd *cobra.Command, args []string) {
		print.PendingStatusEvent(os.Stdout, "Launching Dapr dashboard service.")

		if kubernetesMode {
			print.InfoStatusEvent(os.Stdout, "Launching Dapr dashboard service in kubernetes cluster")
			// handle logic for spinning up dapr dashboard in kubernetes cluster.
		} else {
			print.InfoStatusEvent(os.Stdout, "Launching Dapr dashboard in standalone mode")
			// handle logic for spinning up dapr dashboard in standalone mode.
		}
	},
}

func init() {
 	DashboardCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Deploy Dapr dashboard to a Kubernetes cluster")
 	DashboardCmd.Flags().BoolVarP(&open, "open", "o", false, "Open Dapr dashboard in a browser")
	RootCmd.AddCommand(DashboardCmd)
}