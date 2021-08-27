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

var (
	componentsName         string
	componentsOutputFormat string
)

var ComponentsCmd = &cobra.Command{
	Use:   "components",
	Short: "List all Dapr components. Supported platforms: Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			err := kubernetes.PrintComponents(componentsName, componentsOutputFormat)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}
		}
	},
	Example: `
# List Kubernetes components
dapr components -k
`,
}

func init() {
	ComponentsCmd.Flags().StringVarP(&componentsName, "name", "n", "", "The components name to be printed (optional)")
	ComponentsCmd.Flags().StringVarP(&componentsOutputFormat, "output", "o", "list", "Output format (options: json or yaml or list)")
	ComponentsCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr components in a Kubernetes cluster")
	ComponentsCmd.Flags().BoolP("help", "h", false, "Print this help message")
	ComponentsCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(ComponentsCmd)
}
