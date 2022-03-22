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
	"os"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	configurationName         string
	configurationOutputFormat string
)

var ConfigurationsCmd = &cobra.Command{
	Use:   "configurations",
	Short: "List all Dapr configurations. Supported platforms: Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			print.WarningStatusEvent(os.Stdout, "In future releases, this command will only query the \"default\" namespace by default. Please use the -n (namespace) flag, for specific namespace, or -A (all-namespaces) flag for all namespaces.")
			if allNamespaces {
				resourceNamespace = meta_v1.NamespaceAll
			} else if resourceNamespace == "" {
				resourceNamespace = meta_v1.NamespaceAll
				print.WarningStatusEvent(os.Stdout, "From next release(or after 2 releases), behavior can be changed to query only \"default\" namespace.")
			}
			if configurationName != "" {
				print.WarningStatusEvent(os.Stdout, "From next release(or after 2 releases), behavior can be changed to treat it as \"namespace\".")
			}
			err := kubernetes.PrintConfigurations(configurationName, resourceNamespace, configurationOutputFormat)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}
		}
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		kubernetes.CheckForCertExpiry()
	},
	Example: `
# List all namespace Dapr configurations in Kubernetes mode
dapr configurations -k

# List define namespace Dapr configurations in Kubernetes mode
dapr configurations -k --namespace default

# Print define name Dapr configurations in Kubernetes mode
dapr configurations -k -n target

# List all namespaces Dapr configurations in Kubernetes mode
dapr configurations -k --all-namespaces
`,
}

func init() {
	ConfigurationsCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "If true, list all Dapr configurations in all namespaces")
	ConfigurationsCmd.Flags().StringVarP(&configurationName, "name", "n", "", "The configuration name to be printed (optional)")
	ConfigurationsCmd.Flags().StringVarP(&resourceNamespace, "namespace", "", "", "List Define namespace configurations in a Kubernetes cluster")
	ConfigurationsCmd.Flags().StringVarP(&configurationOutputFormat, "output", "o", "list", "Output format (options: json or yaml or list)")
	ConfigurationsCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr configurations in a Kubernetes cluster")
	ConfigurationsCmd.Flags().BoolP("help", "h", false, "Print this help message")
	ConfigurationsCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(ConfigurationsCmd)
}
