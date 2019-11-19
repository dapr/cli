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
	},
	Run: func(cmd *cobra.Command, args []string) {
		print.PendingStatusEvent(os.Stdout, "Making the jump to hyperspace...")

		if kubernetesMode {
			print.InfoStatusEvent(os.Stdout, "Note: this installation is recommended for testing purposes. For production environments, please use Helm \n")
			err := kubernetes.Init()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				return
			}
			print.SuccessStatusEvent(os.Stdout, "Success! Dapr has been installed. To verify, run 'kubectl get pods -w' in your terminal")
		} else {
			dockerNetwork := viper.GetString("network")
			standalone.Uninstall(true, dockerNetwork)
			err := standalone.Init(runtimeVersion, dockerNetwork)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				return
			}
			print.SuccessStatusEvent(os.Stdout, "Success! Dapr is up and running")
		}
	},
}

func init() {
	InitCmd.Flags().BoolVar(&kubernetesMode, "kubernetes", false, "Deploy Dapr to a Kubernetes cluster")
	InitCmd.Flags().StringVarP(&runtimeVersion, "runtime-version", "", "latest", "The version of the Dapr runtime to install. for example: v0.1.0-alpha")
	InitCmd.Flags().String("network", "", "The Docker network on which to deploy the Dapr runtime")

	RootCmd.AddCommand(InitCmd)
}
