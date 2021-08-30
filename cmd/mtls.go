// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/spf13/cobra"
)

var exportPath string

var MTLSCmd = &cobra.Command{ //nolint
	Use:   "mtls",
	Short: "Check if mTLS is enabled. Supported platforms: Kubernetes",
	Example: `
# Check if mTLS is enabled
dapr mtls -k
`,
	Run: func(cmd *cobra.Command, args []string) {
		enabled, err := kubernetes.IsMTLSEnabled()
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("error checking mTLS: %s", err))
			os.Exit(1)
		}

		status := "disabled"
		if enabled {
			status = "enabled"
		}
		fmt.Printf("Mutual TLS is %s in your Kubernetes cluster \n", status) //nolint
	},
}

var ExportCMD = &cobra.Command{ //nolint
	Use:   "export",
	Short: "Export the root CA, issuer cert and key from Kubernetes to local files",
	Example: `
# Export certs to local folder 
dapr mtls export -o ./certs
`,
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

var ExpiryCMD = &cobra.Command{ //nolint
	Use:   "expiry",
	Short: "Checks the expiry of the root certificate",
	Example: `
# Check expiry of Kubernetes certs
dapr mtls expiry
`,
	Run: func(cmd *cobra.Command, args []string) {
		expiry, err := kubernetes.Expiry()
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("error getting root cert expiry: %s", err))
			return
		}

		duration := int(expiry.Sub(time.Now().UTC()).Hours())
		fmt.Printf("Root certificate expires in %v hours. Expiry date: %s", duration, expiry.String()) // nolint
	},
}

func init() {
	MTLSCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Check if mTLS is enabled in a Kubernetes cluster")
	MTLSCmd.Flags().BoolP("help", "h", false, "Print this help message")
	ExportCMD.Flags().StringVarP(&exportPath, "out", "o", ".", "The output directory path to save the certs")
	ExportCMD.Flags().BoolP("help", "h", false, "Print this help message")
	MTLSCmd.MarkFlagRequired("kubernetes")
	MTLSCmd.AddCommand(ExportCMD)
	MTLSCmd.AddCommand(ExpiryCMD)
	RootCmd.AddCommand(MTLSCmd)
}
