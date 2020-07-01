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

	// defaultLocalPort is the default local port used for port forwarding for `dapr dashboard`
	defaultLocalPort = 8080

	// remotePort is the port dapr dashboard pod is listening on
	remotePort = 8080
)

var localPort int

var DashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Starts Dapr dashboard in a Kubernetes cluster",
	Run: func(cmd *cobra.Command, args []string) {
		if port < 0 {
			localPort = defaultLocalPort
		} else {
			localPort = port
		}

		config, err := kubernetes.GetKubeConfig()

		if err != nil {
			print.FailureStatusEvent(os.Stdout, "Failed to initialize kubernetes client")
			return
		}

		// manage termination of port forwarding connection on interrupt
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		defer signal.Stop(signals)

		portForward, err := kubernetes.NewPortForward(
			config,
			meta_v1.NamespaceDefault,
			dashboardSvc,
			defaultHost,
			localPort,
			remotePort,
			false,
		)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, "%s\n", err)
			os.Exit(1)
		}

		// initialize port forwarding
		if err = portForward.Init(); err != nil {
			print.FailureStatusEvent(os.Stdout, "Error in port forwarding: %s\nCheck for `dapr dashboard` running in other terminal sessions, or use the `--port` flag to use a different port.\n", err)
			os.Exit(1)
		}

		// block until interrupt signal is received
		go func() {
			<-signals
			portForward.Stop()
		}()

		// url for dashboard after port forwarding
		var webURL string = fmt.Sprintf("http://%s:%d", defaultHost, localPort)

		print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Dapr dashboard available at:\t%s\n", webURL))

		err = browser.OpenURL(webURL)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, "Failed to start Dapr dashboard in browser automatically")
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Visit %s in your browser to view the dashboard", webURL))
		}

		<-portForward.GetStop()
	},
}

func init() {
	DashboardCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Start Dapr dashboard in local browser")
	DashboardCmd.Flags().IntVarP(&port, "port", "p", defaultLocalPort, "The local port on which to serve dashboard")
	DashboardCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(DashboardCmd)
}
