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

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
)

var (
	logsAppID string
	podName   string
	namespace string
	k8s       bool
)

var LogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Get Dapr sidecar logs for an application. Supported platforms: Kubernetes",
	Example: `
# Get logs of sample app from target pod in custom namespace
dapr logs -k --app-id sample --pod-name target --namespace custom
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := kubernetes.Logs(logsAppID, podName, namespace)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}
		print.SuccessStatusEvent(os.Stdout, "Fetched logs")
		kubernetes.WarnForCertExpiry()
	},
}

func init() {
	LogsCmd.Flags().BoolVarP(&k8s, "kubernetes", "k", true, "Get logs from a Kubernetes cluster")
	LogsCmd.Flags().StringVarP(&logsAppID, "app-id", "a", "", "The application id for which logs are needed")
	LogsCmd.Flags().StringVarP(&podName, "pod-name", "p", "", "The name of the pod in Kubernetes, in case your application has multiple pods (optional)")
	LogsCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "The Kubernetes namespace in which your application is deployed")
	LogsCmd.Flags().BoolP("help", "h", false, "Print this help message")
	LogsCmd.MarkFlagRequired("app-id")
	LogsCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(LogsCmd)
}
