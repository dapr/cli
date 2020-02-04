// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/spf13/cobra"
)

var MTLSCmd = &cobra.Command{
	Use:   "mtls",
	Short: "Check if mTLS is enabled in a Kubernetes cluster",
	Run: func(cmd *cobra.Command, args []string) {
		enabled, err := kubernetes.IsMTLSEnabled()
		if err != nil {
			fmt.Println(fmt.Sprintf("error checking mTLS: %s", err))
			return
		}

		status := "disabled"
		if enabled {
			status = "enabled"
		}
		fmt.Println(fmt.Sprintf("Mutual TLS is %s in your Kubernetes cluster", status))
	},
}

func init() {
	MTLSCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Check if mTLS is enabled in a Kubernetes cluster")
	MTLSCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(MTLSCmd)
}
