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
	Short: "List all Dapr configurations",
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
}

func init() {
	ConfigurtionsCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr configurations in a k8s cluster")
	ConfigurtionsCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(ConfigurtionsCmd)
}
