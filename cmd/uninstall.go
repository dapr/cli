// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var uninstallVersion string
var uninstallKubernetes bool
var uninstallAll bool

// UninstallCmd is a command from removing a Dapr installation
var UninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Removes a Dapr installation",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("network", cmd.Flags().Lookup("network"))
		viper.BindPFlag("install-path", cmd.Flags().Lookup("install-path"))
	},
	Run: func(cmd *cobra.Command, args []string) {

		var err error

		if uninstallKubernetes {
			print.InfoStatusEvent(os.Stdout, "Removing Dapr from your cluster...")
			err = kubernetes.Uninstall(uninstallVersion)
		} else {
			print.InfoStatusEvent(os.Stdout, "Removing Dapr from your machine...")
			dockerNetwork := viper.GetString("network")
			err = standalone.Uninstall(uninstallAll, dockerNetwork)
		}

		err = errors.New("uninstall failed")
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error removing Dapr: %s", err))
		} else {
			print.SuccessStatusEvent(os.Stdout, "Dapr has been removed successfully")
		}
	},
}

func init() {
	UninstallCmd.Flags().BoolVarP(&uninstallKubernetes, "kubernetes", "k", false, "Uninstall Dapr from a Kubernetes cluster")
	UninstallCmd.Flags().BoolVar(&uninstallAll, "all", false, "Removes .dapr directory, Redis, Placement and Zipkin containers")
	UninstallCmd.Flags().String("network", "", "The Docker network from which to remove the Dapr runtime")
	UninstallCmd.Flags().StringVarP(&uninstallVersion, "runtime-version", "", "latest", "The version of the Dapr runtime to uninstall. for example: v0.1.0 (Kubernetes mode only)")
	RootCmd.AddCommand(UninstallCmd)
}
