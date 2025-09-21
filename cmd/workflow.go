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

import "github.com/spf13/cobra"

var (
	workflowNamespace string
)

var WorkflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Workflow management commands",
}

func init() {
	WorkflowCmd.PersistentFlags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Target a Kubernetes dapr installation")
	WorkflowCmd.PersistentFlags().StringVarP(&workflowNamespace, "namespace", "n", "default", "Namespace to perform workflow operation on")
	RootCmd.AddCommand(WorkflowCmd)
}
