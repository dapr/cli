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
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
)

var (
	kubernetesMode    bool
	wait              bool
	timeout           uint
	slimMode          bool
	runtimeVersion    string
	dashboardVersion  string
	allNamespaces     bool
	initNamespace     string
	resourceNamespace string
	enableMTLS        bool
	enableHA          bool
	values            []string
	fromDir           string
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Install Dapr on supported hosting platforms. Supported platforms: Kubernetes and self-hosted",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("network", cmd.Flags().Lookup("network"))
		viper.BindPFlag("image-registry", cmd.Flags().Lookup("image-registry"))
	},
	Example: `
# Initialize Dapr in self-hosted mode
dapr init

# Initialize Dapr in self-hosted mode with a provided docker image registry. Image looked up as <registry-url>/<image>.
# Check docs or README for more information on the format of the image path that is required. 
dapr init --image-registry <registry-url>

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

# Initialize Dapr from a directory (installer-bundle installation) (Preview feature)
dapr init --from-dir <path-to-directory>

# See more at: https://docs.dapr.io/getting-started/
`,
	Run: func(cmd *cobra.Command, args []string) {
		print.PendingStatusEvent(os.Stdout, "Making the jump to hyperspace...")
		imageRegistryFlag := strings.TrimSpace(viper.GetString("image-registry"))

		if kubernetesMode {
			print.InfoStatusEvent(os.Stdout, "Note: To install Dapr using Helm, see here: https://docs.dapr.io/getting-started/install-dapr-kubernetes/#install-with-helm-advanced\n")
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
			config := kubernetes.InitConfiguration{
				Namespace:        initNamespace,
				Version:          runtimeVersion,
				EnableMTLS:       enableMTLS,
				EnableHA:         enableHA,
				Args:             values,
				Wait:             wait,
				Timeout:          timeout,
				ImageRegistryURI: imageRegistryURI,
			}
			err = kubernetes.Init(config)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}
			print.SuccessStatusEvent(os.Stdout, fmt.Sprintf("Success! Dapr has been installed to namespace %s. To verify, run `dapr status -k' in your terminal. To get started, go here: https://aka.ms/dapr-getting-started", config.Namespace))
		} else {
			dockerNetwork := ""
			imageRegistryURI := ""
			if !slimMode {
				dockerNetwork = viper.GetString("network")
				imageRegistryURI = imageRegistryFlag
			}
			// If both --image-registry and --from-dir flags are given, error out saying only one can be given.
			if len(strings.TrimSpace(imageRegistryURI)) != 0 && len(strings.TrimSpace(fromDir)) != 0 {
				print.FailureStatusEvent(os.Stderr, "both --image-registry and --from-dir flags cannot be given at the same time")
				os.Exit(1)
			}
			if len(strings.TrimSpace(fromDir)) != 0 {
				print.WarningStatusEvent(os.Stdout, "Local bundle installation using --from-dir flag is currently a preview feature and is subject to change. It is only available from CLI version 1.7 onwards.")
			}
			if len(imageRegistryURI) != 0 {
				warnForPrivateRegFeat()
			}
			err := standalone.Init(runtimeVersion, dashboardVersion, dockerNetwork, slimMode, imageRegistryURI, fromDir)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}
			print.SuccessStatusEvent(os.Stdout, "Success! Dapr is up and running. To get started, go here: https://aka.ms/dapr-getting-started")
		}
	},
}

func warnForPrivateRegFeat() {
	print.WarningStatusEvent(os.Stdout, "Flag --image-registry is a preview feature and is subject to change.")
}

func init() {
	defaultRuntimeVersion := "latest"
	viper.BindEnv("runtime_version_override", "DAPR_RUNTIME_VERSION")
	runtimeVersionEnv := viper.GetString("runtime_version_override")
	if runtimeVersionEnv != "" {
		defaultRuntimeVersion = runtimeVersionEnv
	}
	defaultDashboardVersion := "latest"
	viper.BindEnv("dashboard_version_override", "DAPR_DASHBOARD_VERSION")
	dashboardVersionEnv := viper.GetString("dashboard_version_override")
	if dashboardVersionEnv != "" {
		defaultDashboardVersion = dashboardVersionEnv
	}
	InitCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Deploy Dapr to a Kubernetes cluster")
	InitCmd.Flags().BoolVarP(&wait, "wait", "", false, "Wait for Kubernetes initialization to complete")
	InitCmd.Flags().UintVarP(&timeout, "timeout", "", 300, "The wait timeout for the Kubernetes installation")
	InitCmd.Flags().BoolVarP(&slimMode, "slim", "s", false, "Exclude placement service, Redis and Zipkin containers from self-hosted installation")
	InitCmd.Flags().StringVarP(&runtimeVersion, "runtime-version", "", defaultRuntimeVersion, "The version of the Dapr runtime to install, for example: 1.0.0")
	InitCmd.Flags().StringVarP(&dashboardVersion, "dashboard-version", "", defaultDashboardVersion, "The version of the Dapr dashboard to install, for example: 1.0.0")
	InitCmd.Flags().StringVarP(&initNamespace, "namespace", "n", "dapr-system", "The Kubernetes namespace to install Dapr in")
	InitCmd.Flags().BoolVarP(&enableMTLS, "enable-mtls", "", true, "Enable mTLS in your cluster")
	InitCmd.Flags().BoolVarP(&enableHA, "enable-ha", "", false, "Enable high availability (HA) mode")
	InitCmd.Flags().String("network", "", "The Docker network on which to deploy the Dapr runtime")
	InitCmd.Flags().StringVarP(&fromDir, "from-dir", "", "", "Use Dapr artifacts from local directory for self-hosted installation")
	InitCmd.Flags().BoolP("help", "h", false, "Print this help message")
	InitCmd.Flags().StringArrayVar(&values, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	InitCmd.Flags().String("image-registry", "", "Custom/Private docker image repository URL")
	RootCmd.AddCommand(InitCmd)
}
