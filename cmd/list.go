package cmd

import (
	"os"

	"github.com/actionscore/cli/pkg/kubernetes"
	"github.com/actionscore/cli/pkg/print"
	"github.com/actionscore/cli/pkg/standalone"
	"github.com/actionscore/cli/utils"
	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Actions",
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			list, err := kubernetes.List()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			table, err := gocsv.MarshalString(list)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			utils.PrintTable(table)
		} else {
			list, err := standalone.List()
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			if len(list) == 0 {
				println("No Actions found.")
				return
			}

			table, err := gocsv.MarshalString(list)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, err.Error())
				os.Exit(1)
			}

			utils.PrintTable(table)
		}
	},
}

func init() {
	ListCmd.Flags().BoolVar(&kubernetesMode, "kubernetes", false, "List all Actions pods in a k8s cluster")
	RootCmd.AddCommand(ListCmd)
}
