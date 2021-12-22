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
	"os"

	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the health status of Dapr services. Supported platforms: Kubernetes",
	Example: `
# Get status of Dapr services from Kubernetes
dapr status -k 
`,
	Run: func(cmd *cobra.Command, args []string) {
		sc, err := kubernetes.NewStatusClient()
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}
		status, err := sc.Status()
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}
		if len(status) == 0 {
			print.FailureStatusEvent(os.Stderr, "No status returned. Is Dapr initialized in your cluster?")
			os.Exit(1)
		}
		table, err := gocsv.MarshalString(status)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}

		utils.PrintTable(table)
	},
}

func init() {
	StatusCmd.Flags().BoolVarP(&k8s, "kubernetes", "k", false, "Show the health status of Dapr services on Kubernetes cluster")
	StatusCmd.Flags().BoolP("help", "h", false, "Print this help message")
	StatusCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(StatusCmd)
}
