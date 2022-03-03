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

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/spf13/cobra"
)

var (
	certificatePasswordFile     string
	caRootCertificateFile       string
	issuerPrivateKeyFile        string
	issuerPublicCertificateFile string
)
var RenewCertificateCmd = &cobra.Command{
	Use:   "renew-cert",
	Short: "Rotates Dapr root certificate of your kubernetes cluster",
	Example: `
# Generates new root and issuer certificates for kubernetest cluster
dapr upgrade renew-cert -k 

# Uses existing private root.key to generate new root and issuer certificates for kubernetest cluster
dapr upgrade renew-cert -k --certificate-password-file myprivatekey.key

# Rotates certificate of your kubernetes cluster with provided ca.cert, issuer.crt and issuer.key file path
dapr upgrade renew-cert -k --ca-root-certificate <ca.crt> --issuer-private-key <issuer.key> --issuer-public-certificate <issuer.crt>

# See more at: https://docs.dapr.io/getting-started/
`,
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			if caRootCertificateFile != "" && issuerPrivateKeyFile != "" && issuerPublicCertificateFile != "" {
				kubernetes.RenewCertificate(caRootCertificateFile, issuerPrivateKeyFile, issuerPublicCertificateFile)
			} else if certificatePasswordFile != "" {
				fmt.Println("Reuse root password to generatre th root.pem file")
			} else {
				fmt.Println("Generate fresh certificate")
			}
		} else {
			fmt.Println("standalone mode")
		}
		fmt.Println("in subcommand renew-certificate")
	},
}

func init() {
	RenewCertificateCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Upgrades/Renews root certificate of Dapr in a Kubernetes cluster")
	RenewCertificateCmd.Flags().StringVarP(&certificatePasswordFile, "certificate-password-file", "", "", "The version of the Dapr runtime to upgrade or downgrade to, for example: 1.0.0")
	RenewCertificateCmd.Flags().StringVarP(&caRootCertificateFile, "ca-root-certificate", "", "", "The version of the Dapr runtime to upgrade or downgrade to, for example: 1.0.0")
	RenewCertificateCmd.Flags().StringVarP(&issuerPrivateKeyFile, "issuer-private-key", "", "", "The version of the Dapr runtime to upgrade or downgrade to, for example: 1.0.0")
	RenewCertificateCmd.Flags().StringVarP(&issuerPublicCertificateFile, "issuer-public-certificate", "", "", "The version of the Dapr runtime to upgrade or downgrade to, for example: 1.0.0")
	UpgradeCmd.AddCommand(RenewCertificateCmd)
}
