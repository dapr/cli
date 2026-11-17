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

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/kit/signals"
)

var TriggerCmd = &cobra.Command{
	Use:     "trigger",
	Aliases: []string{"run"},
	Short:   "Start a run of a workflow schedule now. Accepts a single argument, the schedule name. The next scheduled run does not move.",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		opts, err := scheduleOptions(cmd)
		if err != nil {
			return err
		}

		id, err := workflow.TriggerSchedule(ctx, opts, args[0])
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "%s", err)
			os.Exit(1)
		}

		print.InfoStatusEvent(os.Stdout, "Workflow schedule %q triggered, starting run: %s", args[0], id)
		return nil
	},
}

func init() {
	ScheduleCmd.AddCommand(TriggerCmd)
}
