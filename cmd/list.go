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

	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/dapr/cli/utils"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var outputFormat string

func outputList(list interface{}, length int) {
	if outputFormat == "json" || outputFormat == "yaml" {
		err := utils.PrintDetail(os.Stdout, outputFormat, list)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}
	} else {
		table, err := gocsv.MarshalString(list)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}

		// Standalone mode displays a separate message when no instances are found.
		if !kubernetesMode && length == 0 {
			fmt.Println("No Dapr instances found.")
			return
		}

		utils.PrintTable(table)
	}
}

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Dapr instances. Supported platforms: Kubernetes and self-hosted",
	Example: `
# List Dapr instances in self-hosted mode
dapr list

# List all namespace Dapr instances in Kubernetes mode
dapr list -k

# List define namespace Dapr instances in Kubernetes mode
dapr list -k -n default

# List all namespaces Dapr instances in Kubernetes mode
dapr list -k --all-namespaces
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if outputFormat != "" && outputFormat != "json" && outputFormat != "yaml" && outputFormat != "table" {
			print.FailureStatusEvent(os.Stdout, "An invalid output format was specified.")
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			if allNamespaces {
				resourceNamespace = meta_v1.NamespaceAll
			} else if resourceNamespace == "" {
				resourceNamespace = meta_v1.NamespaceAll
				print.WarningStatusEvent(os.Stdout, "From next release(or after 2 releases), behavior can be changed to query only \"default\" namespace.")
			}

			list, err := kubernetes.List(resourceNamespace)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}

			outputList(list, len(list))
		} else {
			list, err := standalone.List()
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}

			outputList(list, len(list))
		}
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			kubernetes.CheckForCertExpiry()
		}
	},
}

func init() {
	ListCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "If true, list all Dapr pods in all namespaces")
	ListCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr pods in a Kubernetes cluster")
	ListCmd.Flags().StringVarP(&resourceNamespace, "namespace", "n", "", "List define namespace pod in a Kubernetes cluster")
	ListCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "The output format of the list. Valid values are: json, yaml, or table (default)")
	ListCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(ListCmd)
}
