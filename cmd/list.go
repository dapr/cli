// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/dapr/cli/utils"
	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Dapr instances. Supported platforms: Kubernetes and self-hosted",
	Example: `
# List Dapr instances in self-hosted mode
dapr list

# List Dapr instances in Kubernetes mode
dapr list -k
`,
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			list, err := kubernetes.List()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			table, err := gocsv.MarshalString(list)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			utils.PrintTable(table)
		} else {
			list, err := standalone.List()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			if len(list) == 0 {
				fmt.Println("No Dapr instances found.")
				return
			}

			table, err := gocsv.MarshalString(list)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			utils.PrintTable(table)
		}
	},
}

func init() {
	ListCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr pods in a Kubernetes cluster")
	ListCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(ListCmd)
}
