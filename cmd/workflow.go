/*
Copyright 2025 The Dapr Authors
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
	"fmt"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

var (
	workflowNamespace string
	workflowAppID     string
)

var WorkflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Workflow management commands",
}

func getWorkflowAppID(cmd *cobra.Command) (string, error) {
	if cmd.Flags().Changed("app-id") {
		return workflowAppID, nil
	}

	var errRequired = fmt.Errorf("the app ID is required when there are multiple Dapr instances. Please specify it using the --app-id flag")
	var errNotFound = fmt.Errorf("no Dapr instances found. Please ensure that Dapr is running")

	if kubernetesMode {
		list, err := kubernetes.List(workflowNamespace)
		if err != nil {
			return "", err
		}

		if len(list) == 0 {
			return "", errNotFound
		}

		if len(list) != 1 {
			return "", errRequired
		}

		return list[0].AppID, nil
	}

	list, err := standalone.List()
	if err != nil {
		return "", err
	}

	if len(list) == 0 {
		return "", errNotFound
	}

	if len(list) != 1 {
		return "", errRequired
	}

	return list[0].AppID, nil
}

func init() {
	WorkflowCmd.PersistentFlags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Target a Kubernetes dapr installation")
	WorkflowCmd.PersistentFlags().StringVarP(&workflowNamespace, "namespace", "n", "default", "Namespace to perform workflow operation on")
	WorkflowCmd.PersistentFlags().StringVarP(&workflowAppID, "app-id", "a", "", "The app ID owner of the workflow instance")
	RootCmd.AddCommand(WorkflowCmd)
}
