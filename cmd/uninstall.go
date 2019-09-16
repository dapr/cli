package cmd

import (
	"fmt"
	"os"

	"github.com/actionscore/cli/pkg/kubernetes"
	"github.com/actionscore/cli/pkg/print"
	"github.com/spf13/cobra"
)

var uninstallKubernetes bool

// UninstallCmd is a command from removing an Actions installation
var UninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "removes an Actions installation",
	Run: func(cmd *cobra.Command, args []string) {
		print.InfoStatusEvent(os.Stdout, "Removing Actions from your cluster...")
		err := kubernetes.Uninstall()
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error removing Actions: %s", err))
			return
		}

		print.SuccessStatusEvent(os.Stdout, "Actions has been removed successfully")
	},
}

func init() {
	UninstallCmd.Flags().BoolVar(&uninstallKubernetes, "kubernetes", false, "Uninstall Actions from a Kubernetes cluster (required)")
	UninstallCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(UninstallCmd)
}
