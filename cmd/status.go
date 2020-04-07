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

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Shows the Dapr system services (control plane) health status.",
	Run: func(cmd *cobra.Command, args []string) {
		status, err := kubernetes.Status()
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}
		table, err := gocsv.MarshalString(status)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}

		utils.PrintTable(table)
	},
}

func init() {
	StatusCmd.Flags().BoolVarP(&k8s, "kubernetes", "k", true, "only works with a Kubernetes cluster")
	StatusCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(StatusCmd)
}
