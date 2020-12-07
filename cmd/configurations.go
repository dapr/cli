// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"os"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"
)

var ConfigurtionsCmd = &cobra.Command{
	Use:   "configurations",
	Short: "List all Dapr configurations. Supported platforms: Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			configs, err := kubernetes.Configurations()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			table, err := gocsv.MarshalString(configs)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			utils.PrintTable(table)
		}
	},
	Example: `
# List Kubernetes Dapr configurations
dapr configurations -k
`,
}

func init() {
	ConfigurtionsCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr configurations in a Kubernetes cluster")
	ConfigurtionsCmd.Flags().BoolP("help", "h", false, "Print this help message")
	ConfigurtionsCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(ConfigurtionsCmd)
}
