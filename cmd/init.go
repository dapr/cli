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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/dapr/cli/utils"
)

var (
	kubernetesMode    bool
	wait              bool
	timeout           uint
	slimMode          bool
	devMode           bool
	runtimeVersion    string
	dashboardVersion  string
	allNamespaces     bool
	initNamespace     string
	resourceNamespace string
	enableMTLS        bool
	enableHA          bool
	values            []string
	fromDir           string
	containerRuntime  string
	imageVariant      string
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Install Dapr on supported hosting platforms. Supported platforms: Kubernetes and self-hosted",
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("network", cmd.Flags().Lookup("network"))
		viper.BindPFlag("image-registry", cmd.Flags().Lookup("image-registry"))

		runtimeVersion = getConfigurationValue("runtime-version", cmd)
		dashboardVersion = getConfigurationValue("dashboard-version", cmd)
		containerRuntime = getConfigurationValue("container-runtime", cmd)
	},
	Example: `
# Initialize Dapr in self-hosted mode
dapr init

# Initialize Dapr in self-hosted mode with a provided docker image registry. Image looked up as <registry-url>/<image>.
# Check docs or README for more information on the format of the image path that is required.
dapr init --image-registry <registry-url>

# Initialize Dapr in Kubernetes
dapr init -k

# Initialize Dapr in Kubernetes in dev mode
dapr init -k --dev

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

# Initialize dapr with a particular image variant. Allowed values: "mariner"
dapr init --image-variant <variant>

# Initialize Dapr inside a ".dapr" directory present in a non-default location
# Folder .dapr will be created in folder pointed to by <path-to-install-directory>
dapr init --runtime-path <path-to-install-directory>

# See more at: https://docs.dapr.io/getting-started/
`,
	Run: func(cmd *cobra.Command, args []string) {
		print.PendingStatusEvent(os.Stdout, "Making the jump to hyperspace...")
		imageRegistryFlag := strings.TrimSpace(viper.GetString("image-registry"))

		if kubernetesMode {
			print.InfoStatusEvent(os.Stdout, "Note: To install Dapr using Helm, see here: https://docs.dapr.io/getting-started/install-dapr-kubernetes/#install-with-helm-advanced\n")
			imageRegistryURI := ""
			var err error

			if len(strings.TrimSpace(daprRuntimePath)) != 0 {
				print.FailureStatusEvent(os.Stderr, "--runtime-path is only valid for self-hosted mode")
				os.Exit(1)
			}

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
			if err = verifyCustomCertFlags(cmd); err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}

			config := kubernetes.InitConfiguration{
				Namespace:                 initNamespace,
				Version:                   runtimeVersion,
				DashboardVersion:          dashboardVersion,
				EnableMTLS:                enableMTLS,
				EnableHA:                  enableHA,
				EnableDev:                 devMode,
				Args:                      values,
				Wait:                      wait,
				Timeout:                   timeout,
				ImageRegistryURI:          imageRegistryURI,
				ImageVariant:              imageVariant,
				RootCertificateFilePath:   strings.TrimSpace(caRootCertificateFile),
				IssuerCertificateFilePath: strings.TrimSpace(issuerPublicCertificateFile),
				IssuerPrivateKeyFilePath:  strings.TrimSpace(issuerPrivateKeyFile),
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

			if !utils.IsValidContainerRuntime(containerRuntime) {
				print.FailureStatusEvent(os.Stdout, "Invalid container runtime. Supported values are docker and podman.")
				os.Exit(1)
			}
			err := standalone.Init(runtimeVersion, dashboardVersion, dockerNetwork, slimMode, imageRegistryURI, fromDir, containerRuntime, imageVariant, daprRuntimePath)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}
			print.SuccessStatusEvent(os.Stdout, "Success! Dapr is up and running. To get started, go here: https://aka.ms/dapr-getting-started")
		}
	},
}

func verifyCustomCertFlags(cmd *cobra.Command) error {
	ca := cmd.Flags().Lookup("ca-root-certificate")
	issuerKey := cmd.Flags().Lookup("issuer-private-key")
	issuerCert := cmd.Flags().Lookup("issuer-public-certificate")

	if ca.Changed && len(strings.TrimSpace(ca.Value.String())) == 0 {
		return errors.New("non empty value of --ca-root-certificate must be provided")
	}
	if issuerKey.Changed && len(strings.TrimSpace(issuerKey.Value.String())) == 0 {
		return errors.New("non empty value of --issuer-private-key must be provided")
	}
	if issuerCert.Changed && len(strings.TrimSpace(issuerCert.Value.String())) == 0 {
		return errors.New("non empty value of --issuer-public-certificate must be provided")
	}
	return nil
}

func warnForPrivateRegFeat() {
	print.WarningStatusEvent(os.Stdout, "Flag --image-registry is a preview feature and is subject to change.")
}

func init() {
	defaultRuntimeVersion := "latest"
	defaultDashboardVersion := "latest"
	defaultContainerRuntime := string(utils.DOCKER)

	InitCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Deploy Dapr to a Kubernetes cluster")
	InitCmd.Flags().BoolVarP(&devMode, "dev", "", false, "Use Dev mode. Deploy Redis, Zipkin also in the Kubernetes cluster")
	InitCmd.Flags().BoolVarP(&wait, "wait", "", false, "Wait for Kubernetes initialization to complete")
	InitCmd.Flags().UintVarP(&timeout, "timeout", "", 300, "The wait timeout for the Kubernetes installation")
	InitCmd.Flags().BoolVarP(&slimMode, "slim", "s", false, "Exclude placement service, scheduler service, Redis and Zipkin containers from self-hosted installation")
	InitCmd.Flags().StringVarP(&runtimeVersion, "runtime-version", "", defaultRuntimeVersion, "The version of the Dapr runtime to install, for example: 1.0.0")
	InitCmd.Flags().StringVarP(&dashboardVersion, "dashboard-version", "", defaultDashboardVersion, "The version of the Dapr dashboard to install, for example: 0.13.0")
	InitCmd.Flags().StringVarP(&initNamespace, "namespace", "n", "dapr-system", "The Kubernetes namespace to install Dapr in")
	InitCmd.Flags().BoolVarP(&enableMTLS, "enable-mtls", "", true, "Enable mTLS in your cluster")
	InitCmd.Flags().BoolVarP(&enableHA, "enable-ha", "", false, "Enable high availability (HA) mode")
	InitCmd.Flags().String("network", "", "The Docker network on which to deploy the Dapr runtime")
	InitCmd.Flags().StringVarP(&fromDir, "from-dir", "", "", "Use Dapr artifacts from local directory for self-hosted installation")
	InitCmd.Flags().StringVarP(&imageVariant, "image-variant", "", "", "The image variant to use for the Dapr runtime, for example: mariner")
	InitCmd.Flags().BoolP("help", "h", false, "Print this help message")
	InitCmd.Flags().StringArrayVar(&values, "set", []string{}, "set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	InitCmd.Flags().String("image-registry", "", "Custom/private docker image repository URL")
	InitCmd.Flags().StringVarP(&containerRuntime, "container-runtime", "", defaultContainerRuntime, "The container runtime to use. Supported values are docker (default) and podman")
	InitCmd.Flags().StringVarP(&caRootCertificateFile, "ca-root-certificate", "", "", "The root certificate file")
	InitCmd.Flags().StringVarP(&issuerPrivateKeyFile, "issuer-private-key", "", "", "The issuer certificate private key")
	InitCmd.Flags().StringVarP(&issuerPublicCertificateFile, "issuer-public-certificate", "", "", "The issuer certificate")
	InitCmd.MarkFlagsRequiredTogether("ca-root-certificate", "issuer-private-key", "issuer-public-certificate")

	RootCmd.AddCommand(InitCmd)
}

// getConfigurationValue returns the value for a given configuration key.
// The value is retrieved from the following sources, in order:
// Default value
// Environment variable (respecting registered prefixes)
// Command line flag
// Value is returned as a string.
func getConfigurationValue(n string, cmd *cobra.Command) string {
	viper.BindEnv(n)
	viper.BindPFlag(n, cmd.Flags().Lookup(n))
	return viper.GetString(n)
}
