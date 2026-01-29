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

package scheduler

import (
	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/scheduler"
	"github.com/dapr/kit/signals"
)

var DeleteCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"d", "del"},
	Short:   "Delete one or more jobs from scheduler.",
	Long: `Delete one of more jobs from scheduler.
Job names are formatted by their type, app ID, then identifier.
Actor reminders require the actor type, actor ID, then reminder name, separated by /.
Workflow reminders require the app ID, instance ID, then reminder name, separated by /.
Accepts multiple names.
`,
	Args: cobra.MinimumNArgs(1),
	Example: `
dapr scheduler delete app/my-app-id/my-job-name
dapr scheduler delete actor/my-actor-type/my-actor-id/my-reminder-name
dapr scheduler delete workflow/my-app-id/my-instance-id/my-workflow-reminder-name
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()
		opts := scheduler.DeleteOptions{
			SchedulerNamespace: schedulerNamespace,
			KubernetesMode:     kubernetesMode,
			DaprNamespace:      daprNamespace,
		}

		return scheduler.Delete(ctx, opts, args...)
	},
}

func init() {
	SchedulerCmd.AddCommand(DeleteCmd)
}
