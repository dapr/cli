// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"os"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/spf13/cobra"
)

var UpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrades a Dapr control plane installation in a cluster. Supported platforms: Kubernetes",
	Example: `
# Upgrade Dapr in Kubernetes
dapr upgrade -k

# See more at: https://docs.dapr.io/getting-started/
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := kubernetes.Upgrade(kubernetes.UpgradeConfig{
			RuntimeVersion: runtimeVersion,
		})
		if err != nil {
			print.FailureStatusEvent(os.Stdout, "Failed to upgrade Dapr: %s", err)
			return
		}
		print.SuccessStatusEvent(os.Stdout, "Dapr control plane successfully upgraded to version %s. Make sure your deployments are restarted to pick up the latest sidecar version.", runtimeVersion)
	},
}

func init() {
	UpgradeCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Upgrade Dapr in a Kubernetes cluster")
	UpgradeCmd.Flags().StringVarP(&runtimeVersion, "runtime-version", "", "", "The version of the Dapr runtime to upgrade to, for example: 1.0.0")
	UpgradeCmd.Flags().BoolP("help", "h", false, "Print this help message")

	UpgradeCmd.MarkFlagRequired("runtime-version")
	UpgradeCmd.MarkFlagRequired("kubernetes")

	RootCmd.AddCommand(UpgradeCmd)
}
