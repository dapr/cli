package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/dapr/cli/pkg/print"
	"github.com/spf13/cobra"
)

var uninstallKubernetes bool

// UninstallCmd is a command from removing an Dapr installation
var UninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "removes a dapr installation",
	Run: func(cmd *cobra.Command, args []string) {
		print.InfoStatusEvent(os.Stdout, "Removing Dapr from your cluster...")
		if uninstallKubernetes {
			err := kubernetes.Uninstall()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error removing Dapr: %s", err))
				return
			}

		} else {
			err := standalone.Uninstall()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error removing Dapr: %s", err))
				return
			}
		}
		print.SuccessStatusEvent(os.Stdout, "Dapr has been removed successfully")
	},
}

func init() {
	UninstallCmd.Flags().BoolVar(&uninstallKubernetes, "kubernetes", false, "Uninstall Dapr from a Kubernetes cluster")
	RootCmd.AddCommand(UninstallCmd)
}
