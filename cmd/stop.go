package cmd

import (
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

var stopAppID string

var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops a running dapr instance and its associated app",
	Run: func(cmd *cobra.Command, args []string) {
		err := standalone.Stop(stopAppID)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, "failed to stop app id %s: %s", stopAppID, err)
		} else {
			print.SuccessStatusEvent(os.Stdout, "app stopped successfully")
		}
	},
}

func init() {
	StopCmd.Flags().StringVarP(&stopAppID, "app-id", "", "", "app id to stop (standalone mode)")
	RootCmd.AddCommand(StopCmd)
}
