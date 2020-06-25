// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// dashboardSvc is the name of the dashboard service running in cluster
	dashboardSvc = "dapr-dashboard"

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
		config, err := kubernetes.GetKubeConfig()

		if err != nil {
			print.FailureStatusEvent(os.Stdout, "Failed to initialize kubernetes client")
		}

		print.InfoStatusEvent(os.Stdout, "Launching Dapr dashboard in kubernetes cluster")

		err = kubernetes.InitDashboard()
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Failed to initialize dashboard")
			return
		}

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		defer signal.Stop(signals)

		portForward, err := kubernetes.NewPortForward(
			config,
			meta_v1.NamespaceDefault,
			dashboardSvc,
			defaultHost,
			defaultPort,
			defaultPort,
			false,
		)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Failed to initialize port forwarding: %s\n", err)
			os.Exit(1)
		}

		if err = portForward.Init(); err != nil {
			print.FailureStatusEvent(os.Stderr, "Error initializing port forward. Check for `dapr dashboard` running in other terminal sessions")
			os.Exit(1)
		}

		go func() {
			<-signals
			portForward.Stop()
		}()

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

		<-portForward.GetStop()
	},
}

func init() {
	DashboardCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Deploy Dapr dashboard to a Kubernetes cluster")
	DashboardCmd.Flags().BoolVarP(&open, "open", "o", false, "Open Dapr dashboard in a browser")
	DashboardCmd.Flags().IntVarP(&port, "port", "p", defaultPort, "The local port on which to serve dashboard")
	DashboardCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(DashboardCmd)
}
