// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
)

var upgradeRuntimeVersion string

var UpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrades or downgrades a Dapr control plane installation in a cluster. Supported platforms: Kubernetes",
	Example: `
# Upgrade Dapr in Kubernetes
dapr upgrade -k

# See more at: https://docs.dapr.io/getting-started/
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := kubernetes.Upgrade(kubernetes.UpgradeConfig{
			RuntimeVersion: upgradeRuntimeVersion,
			Args:           values,
			Timeout:        timeout,
		})
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Failed to upgrade Dapr: %s", err)
			os.Exit(1)
		}
		print.SuccessStatusEvent(os.Stdout, "Dapr control plane successfully upgraded to version %s. Make sure your deployments are restarted to pick up the latest sidecar version.", upgradeRuntimeVersion)
	},
}

func init() {
	UpgradeCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Upgrade or downgrade Dapr in a Kubernetes cluster")
	UpgradeCmd.Flags().UintVarP(&timeout, "timeout", "", 300, "The timeout for the Kubernetes upgrade")
	UpgradeCmd.Flags().StringVarP(&upgradeRuntimeVersion, "runtime-version", "", "", "The version of the Dapr runtime to upgrade or downgrade to, for example: 1.0.0")
	UpgradeCmd.Flags().BoolP("help", "h", false, "Print this help message")
	UpgradeCmd.Flags().StringArrayVar(&values, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")

	UpgradeCmd.MarkFlagRequired("runtime-version")
	UpgradeCmd.MarkFlagRequired("kubernetes")

	RootCmd.AddCommand(UpgradeCmd)
}
