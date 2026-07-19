/*
Copyright 2026 The Dapr Authors
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
	"github.com/dapr/cli/pkg/standalone"

	v1alpha1 "github.com/dapr/dapr/pkg/apis/mcpserver/v1alpha1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	mcpServersName         string
	mcpServersOutputFormat string
	mcpServersResourcesDir string
)

var MCPServersCmd = &cobra.Command{
	Use:   "mcpservers",
	Short: "List all Dapr MCPServer resources. Supported platforms: Kubernetes and self-hosted",
	PreRun: func(cmd *cobra.Command, args []string) {
		if mcpServersOutputFormat != "list" && mcpServersOutputFormat != "json" && mcpServersOutputFormat != "yaml" {
			print.FailureStatusEvent(os.Stderr, "An invalid output format was specified. Valid values are: json, yaml, or list")
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			if allNamespaces || resourceNamespace == "" {
				resourceNamespace = meta_v1.NamespaceAll
			}
			if err := kubernetes.PrintMCPServers(mcpServersName, resourceNamespace, mcpServersOutputFormat); err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}
			return
		}

		// Self-hosted: walk the resources directory for MCPServer YAML.
		// Defaults to $HOME/.dapr/components when --resources-path is not set;
		// resolution is handled by pkg/standalone.
		err := kubernetes.WriteMCPServers(os.Stdout, func() (*v1alpha1.MCPServerList, error) {
			return standalone.ListMCPServers(mcpServersResourcesDir)
		}, mcpServersName, mcpServersOutputFormat)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			kubernetes.CheckForCertExpiry()
		}
	},
	Example: `
# List all Dapr MCPServer resources in self-hosted mode (reads from ~/.dapr/components/ by default)
dapr mcpservers

# List MCPServer resources from a custom resources directory in self-hosted mode
dapr mcpservers --resources-path ./resources

# Print a specific MCPServer resource in self-hosted mode
dapr mcpservers -n my-mcp-server

# List all Dapr MCPServer resources in Kubernetes mode
dapr mcpservers -k

# List MCPServer resources in a specific namespace
dapr mcpservers -k --namespace default

# List MCPServer resources across all namespaces
dapr mcpservers -k --all-namespaces

# Output as JSON
dapr mcpservers -o json
`,
}

func init() {
	MCPServersCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "If true, list all Dapr MCPServer resources in all namespaces (Kubernetes mode only)")
	MCPServersCmd.Flags().StringVarP(&mcpServersName, "name", "n", "", "The MCPServer name to be printed (optional)")
	MCPServersCmd.Flags().StringVarP(&resourceNamespace, "namespace", "", "", "List MCPServer resources in a specific Kubernetes namespace")
	MCPServersCmd.Flags().StringVarP(&mcpServersOutputFormat, "output", "o", "list", "Output format (options: json or yaml or list)")
	MCPServersCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List Dapr MCPServer resources from a Kubernetes cluster")
	MCPServersCmd.Flags().StringVar(&mcpServersResourcesDir, "resources-path", "", "Self-hosted only: directory to scan for MCPServer YAML resources (defaults to $HOME/.dapr/components)")
	MCPServersCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(MCPServersCmd)
}
