// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
)

var (
	uninstallNamespace  string
	uninstallKubernetes bool
	uninstallAll        bool
)

// UninstallCmd is a command from removing a Dapr installation.
var UninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Dapr runtime. Supported platforms: Kubernetes and self-hosted",
	Example: `
# Uninstall from self-hosted mode
dapr uninstall

# Uninstall from self-hosted mode and remove .dapr directory, Redis, Placement and Zipkin containers
dapr uninstall --all

# Uninstall from Kubernetes
dapr uninstall -k
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("network", cmd.Flags().Lookup("network"))
		viper.BindPFlag("install-path", cmd.Flags().Lookup("install-path"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		if uninstallKubernetes {
			print.InfoStatusEvent(os.Stdout, "Removing Dapr from your cluster...")
			err = kubernetes.Uninstall(uninstallNamespace, uninstallAll, timeout)
		} else {
			print.InfoStatusEvent(os.Stdout, "Removing Dapr from your machine...")
			dockerNetwork := viper.GetString("network")
			err = standalone.Uninstall(uninstallAll, dockerNetwork)
		}

		if err != nil {
			print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Error removing Dapr: %s", err))
		} else {
			print.SuccessStatusEvent(os.Stdout, "Dapr has been removed successfully")
		}
	},
}

func init() {
	UninstallCmd.Flags().BoolVarP(&uninstallKubernetes, "kubernetes", "k", false, "Uninstall Dapr from a Kubernetes cluster")
	UninstallCmd.Flags().UintVarP(&timeout, "timeout", "", 300, "The timeout for the Kubernetes uninstall")
	UninstallCmd.Flags().BoolVar(&uninstallAll, "all", false, "Remove .dapr directory, Redis, Placement and Zipkin containers on local machine, and CRDs on a Kubernetes cluster")
	UninstallCmd.Flags().String("network", "", "The Docker network from which to remove the Dapr runtime")
	UninstallCmd.Flags().StringVarP(&uninstallNamespace, "namespace", "n", "dapr-system", "The Kubernetes namespace to uninstall Dapr from")
	UninstallCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(UninstallCmd)
}
