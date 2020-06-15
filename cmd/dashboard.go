// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

const (
	// defaultHost is the default host used for port forwarding for `dapr dashboard`
	defaultHost = "localhost"

	// defaultPort is the default port used for port forwarding for `dapr dashboard`
	defaultPort = 8080
)

var open bool

var DashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Runs the Dapr dashboard on a Kubernetes cluster",
	Run: func(cmd *cobra.Command, args []string) {
		print.InfoStatusEvent(os.Stdout, "Launching Dapr dashboard service in kubernetes cluster")
		err := kubernetes.InitDashboard()
		if err != nil {
			print.FailureStatusEvent(os.Stdout, "Failed to initialize dashboard")
			return
		}
		print.SuccessStatusEvent(os.Stdout, "Dapr dashboard initialized successfully")

		// get url for dashboard after port forwarding
		var webUrl string = fmt.Sprintf("https://%s:%d", defaultHost, defaultPort)

		print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Dapr dashboard available at:\t%s\n", webUrl))

		if open {
			print.InfoStatusEvent(os.Stdout, "launching Dapr dashboard in browser")

			err := browser.OpenURL(webUrl)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, "Failed to open Dapr dashboard automatically")
				print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Visit %s in your browser to view the dashboard", webUrl))
			}
		}
	},
}

func init() {
	DashboardCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Deploy Dapr dashboard to a Kubernetes cluster")
	DashboardCmd.Flags().BoolVarP(&open, "open", "o", false, "Open Dapr dashboard in a browser")
	DashboardCmd.Flags().IntVarP(&port, "port", "p", defaultPort, "The local port on which to serve dashboard")
	DashboardCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(DashboardCmd)
}
