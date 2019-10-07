package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/dapr/cli/utils"
	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all dapr instances",
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
				fmt.Println("No dapr instances found.")
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
	ListCmd.Flags().BoolVar(&kubernetesMode, "kubernetes", false, "List all Dapr pods in a k8s cluster")
	RootCmd.AddCommand(ListCmd)
}
