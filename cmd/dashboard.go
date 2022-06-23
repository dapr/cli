/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/dapr/cli/utils"
)

const (
	// dashboardSvc is the name of the dashboard service running in cluster.
	dashboardSvc = "dapr-dashboard"

	// defaultHost is the default host used for port forwarding for `dapr dashboard`.
	defaultHost = "localhost"

	// defaultLocalPort is the default local port used for port forwarding for `dapr dashboard`.
	defaultLocalPort = 8080

	// daprSystemNamespace is the namespace "dapr-system" (recommended Dapr install namespace).
	daprSystemNamespace = "dapr-system"

	// defaultNamespace is the default namespace (dapr init -k installation).
	defaultNamespace = "default"

	// remotePort is the port dapr dashboard pod is listening on.
	remotePort = 8080
)

var (
	dashboardNamespace  string
	dashboardHost       string
	dashboardLocalPort  int
	dashboardVersionCmd bool
)

var DashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start Dapr dashboard. Supported platforms: Kubernetes and self-hosted",
	Example: `
# Start dashboard locally
dapr dashboard

# Start dashboard locally in a specified port 
dapr dashboard -p 9999

# Port forward to dashboard in Kubernetes 
dapr dashboard -k 

# Port forward to dashboard in Kubernetes on all addresses in a specified port
dapr dashboard -k -p 9999 -a 0.0.0.0

# Port forward to dashboard in Kubernetes using a port
dapr dashboard -k -p 9999
`,
	Run: func(cmd *cobra.Command, args []string) {
		if dashboardVersionCmd {
			fmt.Println(standalone.GetDashboardVersion())
			os.Exit(0)
		}

		if !utils.IsAddressLegal(dashboardHost) {
			print.FailureStatusEvent(os.Stdout, "Invalid address: %s", dashboardHost)
			os.Exit(1)
		}

		if dashboardLocalPort <= 0 {
			print.FailureStatusEvent(os.Stderr, "Invalid port: %v", dashboardLocalPort)
			os.Exit(1)
		}

		if kubernetesMode {
			config, client, err := kubernetes.GetKubeConfigClient()
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "Failed to initialize kubernetes client: %s", err.Error())
				os.Exit(1)
			}

			// search for dashboard service namespace in order:
			// user-supplied namespace, dapr-system, default.
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

			// if the service is not found, try to search all pods.
			if foundNamespace == "" {
				ok, nspace := kubernetes.CheckPodExists(client, "", nil, dashboardSvc)

				// if the service is found, tell the user to try with the found namespace.
				// if the service is still not found, throw an error.
				if ok {
					print.InfoStatusEvent(os.Stdout, "Dapr dashboard found in namespace: %s. Run dapr dashboard -k -n %s to use this namespace.", nspace, nspace)
				} else {
					print.FailureStatusEvent(os.Stderr, "Failed to find Dapr dashboard in cluster. Check status of dapr dashboard in the cluster.")
				}
				os.Exit(1)
			}

			// manage termination of port forwarding connection on interrupt.
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, os.Interrupt)
			defer signal.Stop(signals)

			portForward, err := kubernetes.NewPortForward(
				config,
				foundNamespace,
				dashboardSvc,
				dashboardHost,
				dashboardLocalPort,
				remotePort,
				false,
			)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "%s\n", err)
				os.Exit(1)
			}

			// initialize port forwarding.
			if err = portForward.Init(); err != nil {
				print.FailureStatusEvent(os.Stderr, "Error in port forwarding: %s\nCheck for `dapr dashboard` running in other terminal sessions, or use the `--port` flag to use a different port.\n", err)
				os.Exit(1)
			}

			// block until interrupt signal is received.
			go func() {
				<-signals
				portForward.Stop()
			}()

			// url for dashboard after port forwarding.
			var webURL string = fmt.Sprintf("http://%s:%d", dashboardHost, dashboardLocalPort) //nolint:nosprintfhostport

			print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Dapr dashboard found in namespace:\t%s", foundNamespace))
			print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Dapr dashboard available at:\t%s\n", webURL))

			err = browser.OpenURL(webURL)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "Failed to start Dapr dashboard in browser automatically")
				print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Visit %s in your browser to view the dashboard", webURL))
			}

			<-portForward.GetStop()
		} else {
			// Standalone mode.
			err := standalone.NewDashboardCmd(dashboardLocalPort).Run()
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "Dapr dashboard not found. Is Dapr installed?")
			}
		}
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			kubernetes.CheckForCertExpiry()
		}
	},
}

func init() {
	DashboardCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Opens Dapr dashboard in local browser via local proxy to Kubernetes cluster")
	DashboardCmd.Flags().BoolVarP(&dashboardVersionCmd, "version", "v", false, "Print the version for Dapr dashboard")
	DashboardCmd.Flags().StringVarP(&dashboardHost, "address", "a", defaultHost, "Address to listen on. Only accepts IP address or localhost as a value")
	DashboardCmd.Flags().IntVarP(&dashboardLocalPort, "port", "p", defaultLocalPort, "The local port on which to serve Dapr dashboard")
	DashboardCmd.Flags().StringVarP(&dashboardNamespace, "namespace", "n", daprSystemNamespace, "The namespace where Dapr dashboard is running")
	DashboardCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(DashboardCmd)
}
