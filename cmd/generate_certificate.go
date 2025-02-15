package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
)

var (
	certificateValidUntil uint
	certificateSavePath   string
)

func GenerateCertificateCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "generate-certificate",
		Short: "Generate new Dapr root certificate and output to console or save to local folder",
		Example: `
# Generate a new certificate and print to console
dapr mtls generate-certificate

# Generate a new certificate and save to local folder
dapr mtls generate-certificate --out ./certs

# Generate a new certificate with specific expiration days from today
dapr mtls generate-certificate --valid-until <no of days>

# Generate a new certificate with specific expiration days from today and save to local folder
dapr mtls generate-certificate --valid-until <no of days> --out ./certs
`,
		Run: func(cmd *cobra.Command, args []string) {
			rootCertBytes, issuerCertBytes, issuerKeyBytes, err := kubernetes.GenerateNewCertificates(
				time.Hour*time.Duration(certificateValidUntil*24), //nolint:gosec
				"")
			if err != nil {
				print.FailureStatusEvent(os.Stderr, fmt.Sprintf("error generating cert: %s", err))
				os.Exit(1)
			}

			savePathSet := cmd.Flags().Lookup("out").Changed

			if savePathSet {
				_, err := os.Stat(certificateSavePath)

				if os.IsNotExist(err) {
					errDir := os.MkdirAll(certificateSavePath, 0o755)
					if errDir != nil {
						print.FailureStatusEvent(os.Stderr, fmt.Sprintf("error creating directory: %s", err))
						os.Exit(1)
					}
				}
				err = os.WriteFile(filepath.Join(certificateSavePath, "ca.crt"), rootCertBytes, 0o600)
				if err != nil {
					print.FailureStatusEvent(os.Stderr, fmt.Sprintf("error writing ca.crt: %s", err))
					os.Exit(1)
				}

				err = os.WriteFile(filepath.Join(certificateSavePath, "issuer.crt"), issuerCertBytes, 0o600)
				if err != nil {
					print.FailureStatusEvent(os.Stderr, fmt.Sprintf("error writing issuer.crt: %s", err))
					os.Exit(1)
				}

				err = os.WriteFile(filepath.Join(certificateSavePath, "issuer.key"), issuerKeyBytes, 0o600)
				if err != nil {
					print.FailureStatusEvent(os.Stderr, fmt.Sprintf("error writing issuer.key: %s", err))
					os.Exit(1)
				}
				print.InfoStatusEvent(os.Stdout, "Generated new certificates and saved to %s", certificateSavePath)
				print.SuccessStatusEvent(os.Stdout, "CA Root Certificate: %s", filepath.Join(certificateSavePath, "ca.crt"))
				print.SuccessStatusEvent(os.Stdout, "Issuer Certificate: %s", filepath.Join(certificateSavePath, "issuer.crt"))
				print.SuccessStatusEvent(os.Stdout, "Issuer Key: %s", filepath.Join(certificateSavePath, "issuer.key"))
			} else {
				print.InfoStatusEvent(os.Stdout, "Generated new certificates")

				print.SuccessStatusEvent(os.Stdout, "CA Root Certificate: ca.crt")
				fmt.Println(string(rootCertBytes))

				print.SuccessStatusEvent(os.Stdout, "Issuer Certificate: issuer.crt")
				fmt.Println(string(issuerCertBytes))

				print.SuccessStatusEvent(os.Stdout, "Issuer Key: issuer.key")
				fmt.Println(string(issuerKeyBytes))
			}
		},
	}

	command.Flags().UintVarP(&certificateValidUntil, "valid-until", "", 365, "Max days before certificate expires")
	command.Flags().StringVarP(&certificateSavePath, "out", "o", ".", "The output directory path to save the certs")
	return command
}
