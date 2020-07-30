// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var kubernetesMode bool
var slimMode bool
var runtimeVersion string

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Setup dapr in Kubernetes or Standalone modes",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("network", cmd.Flags().Lookup("network"))
		viper.BindPFlag("install-path", cmd.Flags().Lookup("install-path"))
		viper.BindPFlag("redis-host", cmd.Flags().Lookup("redis-host"))

	},
	Run: func(cmd *cobra.Command, args []string) {
		print.PendingStatusEvent(os.Stdout, "Making the jump to hyperspace...")

		if kubernetesMode {
			print.InfoStatusEvent(os.Stdout, "Note: this installation is recommended for testing purposes. For production environments, please use Helm \n")
			err := kubernetes.Init(fmt.Sprintf("v%s", runtimeVersion))
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				return
			}
			print.SuccessStatusEvent(os.Stdout, "Success! Dapr has been installed. To verify, run 'kubectl get pods -w' or 'dapr status -k' in your terminal. To get started, go here: https://aka.ms/dapr-getting-started")
		} else {
			dockerNetwork := ""
			if !slimMode {
				dockerNetwork = viper.GetString("network")
			}
			redisHost := viper.GetString("redis-host")
			err := standalone.Init(runtimeVersion, dockerNetwork, redisHost, slimMode)
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
	InitCmd.Flags().BoolVarP(&slimMode, "slim", "s", false, "Initialize dapr in self-hosted mode without placement, redis and zipkin containers.")
	InitCmd.Flags().StringVarP(&runtimeVersion, "runtime-version", "", "latest", "The version of the Dapr runtime to install. for example: v0.1.0")
	InitCmd.Flags().String("network", "", "The Docker network on which to deploy the Dapr runtime")
	InitCmd.Flags().String("redis-host", "localhost", "The host on which the Redis service resides")

	RootCmd.AddCommand(InitCmd)
}
