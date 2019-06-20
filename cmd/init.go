package cmd

import (
	"os"

	"github.com/actionscore/cli/pkg/kubernetes"
	"github.com/actionscore/cli/pkg/print"
	"github.com/actionscore/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

var kubernetesMode bool

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Setup Actions in Kubernetes or Standalone modes",
	Run: func(cmd *cobra.Command, args []string) {
		print.PendingStatusEvent(os.Stdout, "Making the jump to hyperspace...")

		if kubernetesMode {
			err := kubernetes.Init()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				return
			}
		} else {
			err := standalone.Init()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				return
			}
		}

		print.SuccessStatusEvent(os.Stdout, "Success! Get ready to rumble")
	},
}

func init() {
	InitCmd.Flags().BoolVar(&kubernetesMode, "kubernetes", false, "Deploy Actions to a Kubernetes cluster")
	RootCmd.AddCommand(InitCmd)
}
