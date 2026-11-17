/*
Copyright 2026 The Dapr Authors
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

package schedule

import (
	"os"

	"github.com/spf13/cobra"

	workflowcmd "github.com/dapr/cli/cmd/workflow"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/kit/signals"
)

var listOutputFormat *string

var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List the workflow schedules of an app, with when they run next and how their last run went.",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		opts, err := scheduleOptions(cmd)
		if err != nil {
			return err
		}

		list, err := workflow.ListSchedules(ctx, opts)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "%s", err)
			os.Exit(1)
		}
		if len(list) == 0 {
			print.FailureStatusEvent(os.Stderr, "No workflow schedule found in namespace %q for app ID %q", opts.Namespace, opts.AppID)
			return nil
		}
		return printSchedules(*listOutputFormat, list)
	},
}

func init() {
	listOutputFormat = workflowcmd.OutputFunc(ListCmd)
	ScheduleCmd.AddCommand(ListCmd)
}
