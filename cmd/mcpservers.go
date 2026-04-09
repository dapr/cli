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

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	mcpserversName         string
	mcpserversOutputFormat string
)

var McpserversCmd = &cobra.Command{
	Use:   "mcpservers",
	Short: "List all Dapr MCPServer resources. Supported platforms: Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			if allNamespaces {
				resourceNamespace = meta_v1.NamespaceAll
			} else if resourceNamespace == "" {
				resourceNamespace = meta_v1.NamespaceAll
			}
			err := kubernetes.PrintMCPServers(mcpserversName, resourceNamespace, mcpserversOutputFormat)
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
# List all Dapr MCPServer resources in Kubernetes mode
dapr mcpservers -k

# List MCPServer resources in a specific namespace
dapr mcpservers -k --namespace default

# Print a specific MCPServer resource
dapr mcpservers -k -n my-mcp-server

# List MCPServer resources across all namespaces
dapr mcpservers -k --all-namespaces

# Output as JSON
dapr mcpservers -k -o json
`,
}

func init() {
	McpserversCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "If true, list all Dapr MCPServer resources in all namespaces")
	McpserversCmd.Flags().StringVarP(&mcpserversName, "name", "n", "", "The MCPServer name to be printed (optional)")
	McpserversCmd.Flags().StringVarP(&resourceNamespace, "namespace", "", "", "List MCPServer resources in a specific Kubernetes namespace")
	McpserversCmd.Flags().StringVarP(&mcpserversOutputFormat, "output", "o", "list", "Output format (options: json or yaml or list)")
	McpserversCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr MCPServer resources in a Kubernetes cluster")
	McpserversCmd.Flags().BoolP("help", "h", false, "Print this help message")
	McpserversCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(McpserversCmd)
}
