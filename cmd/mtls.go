// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/spf13/cobra"
)

var exportPath string

var MTLSCmd = &cobra.Command{
	Use:   "mtls",
	Short: "Check if mTLS is enabled in a Kubernetes cluster",
	Run: func(cmd *cobra.Command, args []string) {
		enabled, err := kubernetes.IsMTLSEnabled()
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("error checking mTLS: %s", err))
			return
		}

		status := "disabled"
		if enabled {
			status = "enabled"
		}
		fmt.Printf("Mutual TLS is %s in your Kubernetes cluster \n", status)
	},
}

var ExportCMD = &cobra.Command{
	Use:   "export",
	Short: "Export the root CA, issuer cert and key from Kubernetes to local files",
	Run: func(cmd *cobra.Command, args []string) {
		err := kubernetes.ExportTrustChain(exportPath)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("error exporting trust chain certs: %s", err))
			return
		}

		dir, _ := filepath.Abs(exportPath)
		print.SuccessStatusEvent(os.Stdout, fmt.Sprintf("Trust certs successfully exported to %s", dir))
	},
}

func init() {
	MTLSCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Check if mTLS is enabled in a Kubernetes cluster")
	ExportCMD.Flags().StringVarP(&exportPath, "out", "o", ".", "Output directory path to save the certs")
	MTLSCmd.MarkFlagRequired("kubernetes")
	MTLSCmd.AddCommand(ExportCMD)
	RootCmd.AddCommand(MTLSCmd)
}
