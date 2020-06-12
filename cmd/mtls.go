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
			fmt.Printf("error checking mTLS: %s \n", err)
			return
		}

		status := "disabled"
		if enabled {
			status = "enabled"
		}
		fmt.Printf("Mutual TLS is %s in your Kubernetes cluster \n", status)
	},
}

func init() {
	MTLSCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Check if mTLS is enabled in a Kubernetes cluster")
	MTLSCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(MTLSCmd)
}
