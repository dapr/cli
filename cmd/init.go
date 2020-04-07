// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"os"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var kubernetesMode bool
var runtimeVersion string

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Setup dapr in Kubernetes or Standalone modes",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("network", cmd.Flags().Lookup("network"))
		viper.BindPFlag("install-path", cmd.Flags().Lookup("install-path"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		print.PendingStatusEvent(os.Stdout, "Making the jump to hyperspace...")

		installLocation := viper.GetString("install-path")
		if kubernetesMode {
			print.InfoStatusEvent(os.Stdout, "Note: this installation is recommended for testing purposes. For production environments, please use Helm \n")
			err := kubernetes.Init(runtimeVersion)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				return
			}
			print.SuccessStatusEvent(os.Stdout, "Success! Dapr has been installed. To verify, run 'kubectl get pods -w' in your terminal. To get started, go here: https://aka.ms/dapr-getting-started")
		} else {
			dockerNetwork := viper.GetString("network")
			standalone.Uninstall(true, dockerNetwork)
			err := standalone.Init(runtimeVersion, dockerNetwork, installLocation)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				return
			}
			print.SuccessStatusEvent(os.Stdout, "Success! Dapr is up and running. To get started, go here: https://aka.ms/dapr-getting-started")
		}
	},
}

func init() {
	InitCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Deploy Dapr to a Kubernetes cluster")
	InitCmd.Flags().StringVarP(&runtimeVersion, "runtime-version", "", "latest", "The version of the Dapr runtime to install. for example: v0.1.0-alpha")
	InitCmd.Flags().String("network", "", "The Docker network on which to deploy the Dapr runtime")
	InitCmd.Flags().String("install-path", "", "The optional location to install Dapr to.  The default is /usr/local/bin for Linux/Mac and C:\\dapr for Windows")

	RootCmd.AddCommand(InitCmd)
}
