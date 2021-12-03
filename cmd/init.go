// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
)

var (
	kubernetesMode   bool
	wait             bool
	timeout          uint
	slimMode         bool
	runtimeVersion   string
	dashboardVersion string
	initNamespace    string
	enableMTLS       bool
	enableHA         bool
	values           []string
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Install Dapr on supported hosting platforms. Supported platforms: Kubernetes and self-hosted",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("network", cmd.Flags().Lookup("network"))
	},
	Example: `
# Initialize Dapr in self-hosted mode
dapr init

# Initialize Dapr in Kubernetes
dapr init -k

# Initialize Dapr in Kubernetes and wait for the installation to complete (default timeout is 300s/5m)
dapr init -k --wait --timeout 600

# Initialize particular Dapr runtime in self-hosted mode
dapr init --runtime-version 0.10.0

# Initialize particular Dapr runtime in Kubernetes
dapr init -k --runtime-version 0.10.0

# Initialize Dapr in slim self-hosted mode
dapr init -s

# See more at: https://docs.dapr.io/getting-started/
`,
	Run: func(cmd *cobra.Command, args []string) {
		print.PendingStatusEvent(os.Stdout, "Making the jump to hyperspace...")

		if kubernetesMode {
			print.InfoStatusEvent(os.Stdout, "Note: To install Dapr using Helm, see here: https://docs.dapr.io/getting-started/install-dapr-kubernetes/#install-with-helm-advanced\n")

			config := kubernetes.InitConfiguration{
				Namespace:  initNamespace,
				Version:    runtimeVersion,
				EnableMTLS: enableMTLS,
				EnableHA:   enableHA,
				Args:       values,
				Wait:       wait,
				Timeout:    timeout,
			}
			err := kubernetes.Init(config)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}
			print.SuccessStatusEvent(os.Stdout, fmt.Sprintf("Success! Dapr has been installed to namespace %s. To verify, run `dapr status -k' in your terminal. To get started, go here: https://aka.ms/dapr-getting-started", config.Namespace))
		} else {
			dockerNetwork := ""
			if !slimMode {
				dockerNetwork = viper.GetString("network")
			}
			err := standalone.Init(runtimeVersion, dashboardVersion, dockerNetwork, slimMode)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}
			print.SuccessStatusEvent(os.Stdout, "Success! Dapr is up and running. To get started, go here: https://aka.ms/dapr-getting-started")
		}
	},
}

func init() {
	defaultRuntimeVersion := "latest"
	viper.BindEnv("runtime_version_override", "DAPR_RUNTIME_VERSION")
	runtimeVersionEnv := viper.GetString("runtime_version_override")
	if runtimeVersionEnv != "" {
		defaultRuntimeVersion = runtimeVersionEnv
	}
	InitCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Deploy Dapr to a Kubernetes cluster")
	InitCmd.Flags().BoolVarP(&wait, "wait", "", false, "Wait for Kubernetes initialization to complete")
	InitCmd.Flags().UintVarP(&timeout, "timeout", "", 300, "The wait timeout for the Kubernetes installation")
	InitCmd.Flags().BoolVarP(&slimMode, "slim", "s", false, "Exclude placement service, Redis and Zipkin containers from self-hosted installation")
	InitCmd.Flags().StringVarP(&runtimeVersion, "runtime-version", "", defaultRuntimeVersion, "The version of the Dapr runtime to install, for example: 1.0.0")
	InitCmd.Flags().StringVarP(&dashboardVersion, "dashboard-version", "", "latest", "The version of the Dapr dashboard to install, for example: 1.0.0")
	InitCmd.Flags().StringVarP(&initNamespace, "namespace", "n", "dapr-system", "The Kubernetes namespace to install Dapr in")
	InitCmd.Flags().BoolVarP(&enableMTLS, "enable-mtls", "", true, "Enable mTLS in your cluster")
	InitCmd.Flags().BoolVarP(&enableHA, "enable-ha", "", false, "Enable high availability (HA) mode")
	InitCmd.Flags().String("network", "", "The Docker network on which to deploy the Dapr runtime")
	InitCmd.Flags().BoolP("help", "h", false, "Print this help message")
	InitCmd.Flags().StringArrayVar(&values, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	RootCmd.AddCommand(InitCmd)
}
