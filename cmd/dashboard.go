// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

const (
	// dashboardSvc is the name of the dashboard service running in cluster
	dashboardSvc = "dapr-dashboard"

	// defaultHost is the default host used for port forwarding for `dapr dashboard`
	defaultHost = "localhost"

	// defaultLocalPort is the default local port used for port forwarding for `dapr dashboard`
	defaultLocalPort = 8080

	// daprSystemNamespace is the namespace "dapr-system" (recommended Dapr install namespace)
	daprSystemNamespace = "dapr-system"

	// defaultNamespace is the default namespace (dapr init -k installation)
	defaultNamespace = "default"

	// remotePort is the port dapr dashboard pod is listening on
	remotePort = 8080
)

var dashboardNamespace string
var localPort int

var DashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start Dapr dashboard",
	Run: func(cmd *cobra.Command, args []string) {
		if port < 0 {
			localPort = defaultLocalPort
		} else {
			localPort = port
		}

		if kubernetesMode {
			config, client, err := kubernetes.GetKubeConfigClient()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, "Failed to initialize kubernetes client: %s", err.Error())
				os.Exit(1)
			}

			// search for dashboard service namespace in order:
			// user-supplied namespace, dapr-system, default
			namespaces := []string{dashboardNamespace}
			if dashboardNamespace != daprSystemNamespace {
				namespaces = append(namespaces, daprSystemNamespace)
			}
			if dashboardNamespace != defaultNamespace {
				namespaces = append(namespaces, defaultNamespace)
			}

			foundNamespace := ""
			for _, namespace := range namespaces {
				ok, _ := kubernetes.CheckPodExists(client, namespace, nil, dashboardSvc)
				if ok {
					foundNamespace = namespace
					break
				}
			}

			// if the service is not found, try to search all pods
			if foundNamespace == "" {
				ok, nspace := kubernetes.CheckPodExists(client, "", nil, dashboardSvc)

				// if the service is found, tell the user to try with the found namespace
				// if the service is still not found, throw an error
				if ok {
					print.InfoStatusEvent(os.Stdout, "Dapr dashboard found in namespace: %s. Run dapr dashboard -k -n %s to use this namespace.", nspace, nspace)

				} else {
					print.FailureStatusEvent(os.Stdout, "Failed to find Dapr dashboard in cluster. Check status of dapr dashboard in the cluster.")
				}
				os.Exit(1)
			}

			// manage termination of port forwarding connection on interrupt
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, os.Interrupt)
			defer signal.Stop(signals)

			portForward, err := kubernetes.NewPortForward(
				config,
				foundNamespace,
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

			print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Dapr dashboard found in namespace:\t%s", foundNamespace))
			print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Dapr dashboard available at:\t%s\n", webURL))

			err = browser.OpenURL(webURL)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, "Failed to start Dapr dashboard in browser automatically")
				print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Visit %s in your browser to view the dashboard", webURL))
			}

			<-portForward.GetStop()
		} else {
			// Standalone mode

			// Use the default binary install location
			dashboardPath := standalone.DefaultDaprBinPath()

			binaryName := "dashboard"
			if runtime.GOOS == "windows" {
				binaryName = "dashboard.exe"
			}

			// Construct command to run dashboard
			cmdDashboardStandalone := &exec.Cmd{
				Path:   filepath.Join(dashboardPath, binaryName),
				Dir:    dashboardPath,
				Stdout: os.Stdout,
			}

			err := cmdDashboardStandalone.Run()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, "Dapr dashboard not found. Is Dapr installed?")
			}
		}
	},
}

func init() {
	DashboardCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Start Dapr dashboard in local browser")
	DashboardCmd.Flags().IntVarP(&port, "port", "p", defaultLocalPort, "The local port on which to serve dashboard")
	DashboardCmd.Flags().StringVarP(&dashboardNamespace, "namespace", "n", daprSystemNamespace, "The namespace where Dapr dashboard is running")
	RootCmd.AddCommand(DashboardCmd)
}
