package cmd

import (
	"fmt"
	"os"

	"github.com/actionscore/cli/pkg/kubernetes"
	"github.com/actionscore/cli/pkg/print"
	"github.com/spf13/cobra"
)

// UninstallCmd is a command from removing Actions from a Kubernetes cluster
var UninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "removes Actions from a Kubernetes cluster",
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
	RootCmd.AddCommand(UninstallCmd)
}
