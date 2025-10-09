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

var DeleteAllCmd = &cobra.Command{
	Use:     "delete-all",
	Aliases: []string{"da", "delall"},
	Short: `Delete all scheduled jobs in the specified namespace of a particular filter.
Accepts a single key as an argument. Deletes all jobs which match the filter key.
`,
	Args: cobra.ExactArgs(1),
	Example: `
dapr scheduler delete-all all
dapr scheduler delete-all app
dapr scheduler delete-all app/my-app-id
dapr scheduler delete-all actor/my-actor-type
dapr scheduler delete-all actor/my-actor-type/my-actor-id
dapr scheduler delete-all workflow
dapr scheduler delete-all workflow/my-app-id
dapr scheduler delete-all workflow/my-app-id/my-workflow-id
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()
		opts := scheduler.DeleteOptions{
			SchedulerNamespace: schedulerNamespace,
			KubernetesMode:     kubernetesMode,
			DaprNamespace:      daprNamespace,
		}

		return scheduler.DeleteAll(ctx, opts, args[0])
	},
}

func init() {
	SchedulerCmd.AddCommand(DeleteAllCmd)
}
