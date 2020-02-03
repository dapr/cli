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

var ComponentsCmd = &cobra.Command{
	Use:   "components",
	Short: "List all Dapr components",
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			components, err := kubernetes.Components()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			table, err := gocsv.MarshalString(components)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			utils.PrintTable(table)
		}
	},
}

func init() {
	ComponentsCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr components in a k8s cluster")
	ComponentsCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(ComponentsCmd)
}
