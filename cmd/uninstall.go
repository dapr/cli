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
	"github.com/spf13/cobra"
)

var uninstallKubernetes bool
var uninstallAll bool

// UninstallCmd is a command from removing an Dapr installation
var UninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "removes a dapr installation",
	Run: func(cmd *cobra.Command, args []string) {
		print.InfoStatusEvent(os.Stdout, "Removing Dapr from your cluster...")

		var err error

		if uninstallKubernetes {
			err = kubernetes.Uninstall()
		} else {
			err = standalone.Uninstall(uninstallAll)
		}

		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error removing Dapr: %s", err))
		} else {
			print.SuccessStatusEvent(os.Stdout, "Dapr has been removed successfully")
		}
	},
}

func init() {
	UninstallCmd.Flags().BoolVar(&uninstallKubernetes, "kubernetes", false, "Uninstall Dapr from a Kubernetes cluster")
	UninstallCmd.Flags().BoolVar(&uninstallAll, "all", false, "Remove the redis container as well")
	RootCmd.AddCommand(UninstallCmd)
}
