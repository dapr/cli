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
	configurationName         string
	configurationOutputFormat string
)

var ConfigurationsCmd = &cobra.Command{
	Use:   "configurations",
	Short: "List all Dapr configurations. Supported platforms: Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			err := kubernetes.PrintConfigurations(configurationName, configurationOutputFormat)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}
		}
	},
	Example: `
# List Kubernetes Dapr configurations
dapr configurations -k
`,
}

func init() {
	ConfigurationsCmd.Flags().StringVarP(&configurationName, "name", "n", "", "The configuration name to be printed (optional)")
	ConfigurationsCmd.Flags().StringVarP(&configurationOutputFormat, "output", "o", "list", "Output format (options: json or yaml or list)")
	ConfigurationsCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr configurations in a Kubernetes cluster")
	ConfigurationsCmd.Flags().BoolP("help", "h", false, "Print this help message")
	ConfigurationsCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(ConfigurationsCmd)
}
