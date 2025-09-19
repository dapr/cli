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
	schedulerSchedulerNamespace string
	schedulerNamespace          string
)

var SchedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Scheduler management commands",
}

func init() {
	RootCmd.AddCommand(SchedulerCmd)
	SchedulerCmd.PersistentFlags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr pods in a Kubernetes cluster")
	SchedulerCmd.PersistentFlags().StringVarP(&schedulerNamespace, "namespace", "n", "", "Kubernetes namespace to list Dapr apps from. If not specified, uses all namespaces")
	SchedulerCmd.PersistentFlags().StringVar(&schedulerSchedulerNamespace, "scheduler-namespace", "dapr-system", "Kubernetes namespace where the scheduler is deployed")
}
