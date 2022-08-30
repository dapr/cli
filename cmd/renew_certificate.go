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
	"time"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

var (
	privateKey                  string
	caRootCertificateFile       string
	issuerPrivateKeyFile        string
	issuerPublicCertificateFile string
	validUntil                  uint
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
dapr mtls renew-certificate -k --private-key myprivatekey.key --valid-until <no of days>

# Rotates certificate of your kubernetes cluster with provided ca.cert, issuer.crt and issuer.key file path
dapr mtls renew-certificate -k --ca-root-certificate <root.pem> --issuer-private-key <issuer.key> --issuer-public-certificate <issuer.pem> --restart

# See more at: https://docs.dapr.io/getting-started/
`,

		Run: func(cmd *cobra.Command, args []string) {
			var err error
			pkFlag := cmd.Flags().Lookup("private-key").Changed
			rootcertFlag := cmd.Flags().Lookup("ca-root-certificate").Changed
			issuerKeyFlag := cmd.Flags().Lookup("issuer-private-key").Changed
			issuerCertFlag := cmd.Flags().Lookup("issuer-public-certificate").Changed

			if kubernetesMode {
				print.PendingStatusEvent(os.Stdout, "Starting certificate rotation")
				if rootcertFlag || issuerKeyFlag || issuerCertFlag {
					flagArgsEmpty := checkReqFlagArgsEmpty(caRootCertificateFile, issuerPrivateKeyFile, issuerPublicCertificateFile)
					if flagArgsEmpty {
						err = fmt.Errorf("all required flags for this certificate rotation path, %q, %q and %q are not present",
							"ca-root-certificate", "issuer-private-key", "issuer-public-certificate")
						logErrorAndExit(err)
					}
					print.InfoStatusEvent(os.Stdout, "Using provided certificates")
					err = kubernetes.RenewCertificate(kubernetes.RenewCertificateParams{
						RootCertificateFilePath:   caRootCertificateFile,
						IssuerCertificateFilePath: issuerPublicCertificateFile,
						IssuerPrivateKeyFilePath:  issuerPrivateKeyFile,
						Timeout:                   timeout,
					})
					if err != nil {
						logErrorAndExit(err)
					}
				} else if pkFlag {
					flagArgsEmpty := checkReqFlagArgsEmpty(privateKey)
					if flagArgsEmpty {
						err = fmt.Errorf("%q flag has incorrect value", "privateKey")
						logErrorAndExit(err)
					}
					print.InfoStatusEvent(os.Stdout, "Using password file to generate root certificate")
					err = kubernetes.RenewCertificate(kubernetes.RenewCertificateParams{
						RootPrivateKeyFilePath: privateKey,
						ValidUntil:             time.Hour * time.Duration(validUntil*24),
						Timeout:                timeout,
					})
					if err != nil {
						logErrorAndExit(err)
					}
				} else {
					print.InfoStatusEvent(os.Stdout, "generating fresh certificates")
					err = kubernetes.RenewCertificate(kubernetes.RenewCertificateParams{
						ValidUntil: time.Hour * time.Duration(validUntil*24),
						Timeout:    timeout,
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
				restartControlPlaneService()
				if err != nil {
					print.FailureStatusEvent(os.Stdout, err.Error())
					os.Exit(1)
				}
			}
		},
	}

	command.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Renews root and issuer certificates of Dapr in a Kubernetes cluster")
	command.Flags().StringVarP(&privateKey, "private-key", "", "", "The root.key file which is used to generate root certificate")
	command.Flags().StringVarP(&caRootCertificateFile, "ca-root-certificate", "", "", "The root certificate file")
	command.Flags().StringVarP(&issuerPrivateKeyFile, "issuer-private-key", "", "", "The issuer certificate private key")
	command.Flags().StringVarP(&issuerPublicCertificateFile, "issuer-public-certificate", "", "", "The issuer certificate")
	command.Flags().UintVarP(&validUntil, "valid-until", "", 365, "Max days before certificate expires")
	command.Flags().BoolVarP(&restartDaprServices, "restart", "", false, "Restart Dapr control plane services")
	command.Flags().UintVarP(&timeout, "timeout", "", 300, "The timeout for the certificate renewal")
	command.MarkFlagRequired("kubernetes")
	return command
}

func checkReqFlagArgsEmpty(params ...string) bool {
	for _, val := range params {
		if len(strings.TrimSpace(val)) == 0 {
			return true
		}
	}
	return false
}

func logErrorAndExit(err error) {
	err = fmt.Errorf("certificate rotation failed: %w", err)
	print.FailureStatusEvent(os.Stderr, err.Error())
	os.Exit(1)
}

func restartControlPlaneService() error {
	controlPlaneServices := []string{"deploy/dapr-sentry", "deploy/dapr-operator", "statefulsets/dapr-placement-server"}
	namespace, err := kubernetes.GetDaprNamespace()
	if err != nil {
		print.FailureStatusEvent(os.Stdout, "Failed to fetch Dapr namespace")
	}
	for _, name := range controlPlaneServices {
		print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Restarting %s..", name))
		_, err := utils.RunCmdAndWait("kubectl", "rollout", "restart", name, "-n", namespace)
		if err != nil {
			return fmt.Errorf("error in restarting deployment %s. Error is %w", name, err)
		}
		_, err = utils.RunCmdAndWait("kubectl", "rollout", "status", name, "-n", namespace)
		if err != nil {
			return fmt.Errorf("error in checking status for deployment %s. Error is %w", name, err)
		}
	}
	print.SuccessStatusEvent(os.Stdout, "All control plane services have restarted successfully!")
	return nil
}
