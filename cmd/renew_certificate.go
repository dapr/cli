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
	"time"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

var (
	certificatePasswordFile     string
	caRootCertificateFile       string
	issuerPrivateKeyFile        string
	issuerPublicCertificateFile string
	validUntil                  int
	restartDaprServices         bool
)

func RenewCertificateCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "renew-certificate",
		Short: "Rotates Dapr root certificate of your kubernetes cluster",

		Example: `
# Generates new root and issuer certificates for kubernetes cluster
dapr mtls renew-certificate -k --valid-until <no of days> --restart

# Uses existing private root.key to generate new root and issuer certificates for kubernetes cluster
dapr mtls renew-certificate -k --certificate-password-file myprivatekey.key --valid-until <no of days>

# Rotates certificate of your kubernetes cluster with provided ca.cert, issuer.crt and issuer.key file path
dapr mtls renew-certificate -k --ca-root-certificate <ca.crt> --issuer-private-key <issuer.key> --issuer-public-certificate <issuer.crt> --restart

# See more at: https://docs.dapr.io/getting-started/
`,

		Run: func(cmd *cobra.Command, args []string) {
			if kubernetesMode {
				print.PendingStatusEvent(os.Stdout, "Starting certificate rotation")
				if caRootCertificateFile != "" && issuerPrivateKeyFile != "" && issuerPublicCertificateFile != "" {
					print.InfoStatusEvent(os.Stdout, "Using provided certificates")
					err := kubernetes.RenewCertificate(kubernetes.RenewCertificateParams{
						RootCertificateFilePath:   caRootCertificateFile,
						IssuerCertificateFilePath: issuerPrivateKeyFile,
						IssuerPrivateKeyFilePath:  issuerPublicCertificateFile,
					})
					if err != nil {
						logErrorAndExit(err)
					}
				} else if certificatePasswordFile != "" {
					print.InfoStatusEvent(os.Stdout, "Using password file to generate root certificate")
					err := kubernetes.RenewCertificate(kubernetes.RenewCertificateParams{
						RootPrivateKeyFilePath: certificatePasswordFile,
						ValidUntil:             time.Hour * time.Duration(validUntil*24),
					})
					if err != nil {
						logErrorAndExit(err)
					}
				} else {
					print.InfoStatusEvent(os.Stdout, "generating fresh certificates")
					err := kubernetes.RenewCertificate(kubernetes.RenewCertificateParams{
						ValidUntil: time.Hour * time.Duration(validUntil*24),
					})
					if err != nil {
						logErrorAndExit(err)
					}
				}
			}
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			expiry, err := kubernetes.Expiry()
			if err != nil {
				logErrorAndExit(err)
			}
			print.SuccessStatusEvent(os.Stdout,
				fmt.Sprintf("Certificate rotation is successful! Your new certicate is valid through %s", expiry.Format(time.RFC1123)))

			if restartDaprServices {
				restartControlPlaneService("deploy/dapr-sentry", "deploy/dapr-operator", "statefulsets/dapr-placement-server")
				if err != nil {
					print.FailureStatusEvent(os.Stdout, err.Error())
					os.Exit(1)
				}
			}
		},
	}

	command.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Upgrades/Renews root certificate of Dapr in a Kubernetes cluster")
	command.Flags().StringVarP(&certificatePasswordFile, "certificate-password-file", "", "", "The version of the Dapr runtime to upgrade or downgrade to, for example: 1.0.0")
	command.Flags().StringVarP(&caRootCertificateFile, "ca-root-certificate", "", "", "The version of the Dapr runtime to upgrade or downgrade to, for example: 1.0.0")
	command.Flags().StringVarP(&issuerPrivateKeyFile, "issuer-private-key", "", "", "The version of the Dapr runtime to upgrade or downgrade to, for example: 1.0.0")
	command.Flags().StringVarP(&issuerPublicCertificateFile, "issuer-public-certificate", "", "", "The version of the Dapr runtime to upgrade or downgrade to, for example: 1.0.0")
	command.Flags().IntVarP(&validUntil, "valid-until", "", 365, "Max days before certificate expires")
	command.Flags().BoolVarP(&restartDaprServices, "restart", "", false, "Restart Dapr control plane services")
	command.MarkFlagRequired("kubernetes")
	return command
}

func logErrorAndExit(err error) {
	err = fmt.Errorf("certificate rotation failed %w", err)
	print.FailureStatusEvent(os.Stderr, err.Error())
	os.Exit(1)
}

func restartControlPlaneService(names ...string) error {
	for _, name := range names {
		print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Restarting %s..", name))
		_, err := utils.RunCmdAndWait("kubectl", "rollout", "restart", name, "-n", "dapr-system")
		if err != nil {
			return fmt.Errorf("error in restarting deployment %s. Error is %w", name, err)
		}
		_, err = utils.RunCmdAndWait("kubectl", "rollout", "status", name, "-n", "dapr-system")
		if err != nil {
			return fmt.Errorf("error in checking status for deployment %s. Error is %w", name, err)
		}
	}
	print.SuccessStatusEvent(os.Stdout, "All control plane services have restarted successfully!")
	return nil
}
