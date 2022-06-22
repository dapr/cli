/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
)

var upgradeRuntimeVersion string

var UpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrades or downgrades a Dapr control plane installation in a cluster. Supported platforms: Kubernetes",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("image-registry", cmd.Flags().Lookup("image-registry"))
	},
	Example: `
# Upgrade Dapr in Kubernetes
dapr upgrade -k

# See more at: https://docs.dapr.io/getting-started/
`,
	Run: func(cmd *cobra.Command, args []string) {
		imageRegistryFlag := strings.TrimSpace(viper.GetString("image-registry"))
		imageRegistryURI := ""
		var err error

		if len(imageRegistryFlag) != 0 {
			warnForPrivateRegFeat()
			imageRegistryURI = imageRegistryFlag
		} else {
			imageRegistryURI, err = kubernetes.GetImageRegistry()
		}
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}
		err = kubernetes.Upgrade(kubernetes.UpgradeConfig{
			RuntimeVersion:   upgradeRuntimeVersion,
			Args:             values,
			Timeout:          timeout,
			ImageRegistryURI: imageRegistryURI,
		})
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Failed to upgrade Dapr: %s", err)
			os.Exit(1)
		}
		print.SuccessStatusEvent(os.Stdout, "Dapr control plane successfully upgraded to version %s. Make sure your deployments are restarted to pick up the latest sidecar version.", upgradeRuntimeVersion)
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		kubernetes.CheckForCertExpiry()
	},
}

func init() {
	UpgradeCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Upgrade or downgrade Dapr in a Kubernetes cluster")
	UpgradeCmd.Flags().UintVarP(&timeout, "timeout", "", 300, "The timeout for the Kubernetes upgrade")
	UpgradeCmd.Flags().StringVarP(&upgradeRuntimeVersion, "runtime-version", "", "", "The version of the Dapr runtime to upgrade or downgrade to, for example: 1.0.0")
	UpgradeCmd.Flags().BoolP("help", "h", false, "Print this help message")
	UpgradeCmd.Flags().StringArrayVar(&values, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	UpgradeCmd.Flags().String("image-registry", "", "Custom/Private docker image repository URL")

	UpgradeCmd.MarkFlagRequired("runtime-version")
	UpgradeCmd.MarkFlagRequired("kubernetes")

	RootCmd.AddCommand(UpgradeCmd)
}
