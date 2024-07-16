package cmd

import (
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

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

var (
	subscriptionsAppId string
)

var SubscriptionsCmd = &cobra.Command{
	Use:   "subscriptions",
	Short: "List all Dapr subscriptions. Supported platforms: Supported platforms: Kubernetes and self-hosted",
	PreRun: func(cmd *cobra.Command, args []string) {
		if outputFormat != "" && outputFormat != "json" && outputFormat != "yaml" && outputFormat != "table" {
			print.FailureStatusEvent(os.Stdout, "An invalid output format was specified.")
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if kubernetesMode {
			if allNamespaces {
				resourceNamespace = meta_v1.NamespaceAll
			} else if resourceNamespace == "" {
				resourceNamespace = meta_v1.NamespaceAll
			}
			//do something to get it in k8s
		} else {
			list, err := standalone.Subscriptions(subscriptionsAppId)
			if err != nil {
				return
			}
			outputList(list, len(list))
		}
	},
	Example: `
# List Dapr subscriptions for a given app
dapr components --app-id myapp
`,
}

func init() {
	SubscriptionsCmd.Flags().StringVarP(&subscriptionsAppId, "app-id", "a", "", "The application id to be stopped")
	SubscriptionsCmd.MarkFlagRequired("app-id")
	SubscriptionsCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "The output format of the list. Valid values are: json, yaml, or table (default)")
	RootCmd.AddCommand(SubscriptionsCmd)
}
