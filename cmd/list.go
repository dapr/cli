// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

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

# List Dapr instances in Kubernetes mode
dapr list -k
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if outputFormat != "" && outputFormat != "json" && outputFormat != "yaml" && outputFormat != "table" {
			print.FailureStatusEvent(os.Stdout, "An invalid output format was specified.")
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			list, err := kubernetes.List()
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
}

func init() {
	ListCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr pods in a Kubernetes cluster")
	ListCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "The output format of the list. Valid values are: json, yaml, or table (default)")
	ListCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(ListCmd)
}
