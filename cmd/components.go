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
	componentsName         string
	componentsOutputFormat string
)

var ComponentsCmd = &cobra.Command{
	Use:   "components",
	Short: "List all Dapr components. Supported platforms: Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			if !noNamespaceWarning {
				print.WarningStatusEvent(os.Stdout, "In future releases, this command will only query the \"default\" namespace by default. Please use the --namespace flag for a specific namespace, or the --all-namespaces (-A) flag for all namespaces.")
			}
			if allNamespaces {
				resourceNamespace = meta_v1.NamespaceAll
			} else if resourceNamespace == "" {
				resourceNamespace = meta_v1.NamespaceAll
			}
			err := kubernetes.PrintComponents(componentsName, resourceNamespace, componentsOutputFormat)
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
# List Dapr components in all namespaces in Kubernetes mode
dapr components -k

# List Dapr components in specific namespace in Kubernetes mode
dapr components -k --namespace default

# Print specific Dapr component in Kubernetes mode
dapr components -k -n target

# List Dapr components in all namespaces in Kubernetes mode
dapr components -k --all-namespaces
`,
}

func init() {
	ComponentsCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "If true, list all Dapr components in all namespaces")
	ComponentsCmd.Flags().BoolVar(&noNamespaceWarning, "no-namespace-warning", false, "Disable upcoming namespace warnings in command output")
	ComponentsCmd.Flags().StringVarP(&componentsName, "name", "n", "", "The components name to be printed (optional)")
	ComponentsCmd.Flags().StringVarP(&resourceNamespace, "namespace", "", "", "List all namespace components in a Kubernetes cluster")
	ComponentsCmd.Flags().StringVarP(&componentsOutputFormat, "output", "o", "list", "Output format (options: json or yaml or list)")
	ComponentsCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr components in a Kubernetes cluster")
	ComponentsCmd.Flags().BoolP("help", "h", false, "Print this help message")
	ComponentsCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(ComponentsCmd)
}
