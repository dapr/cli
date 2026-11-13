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

var getOutputFormat *string

var GetCmd = &cobra.Command{
	Use:     "get",
	Aliases: []string{"show"},
	Short:   "Get a workflow schedule. Accepts a single argument, the schedule name.",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		opts, err := scheduleOptions(cmd)
		if err != nil {
			return err
		}

		ws, err := workflow.GetSchedule(ctx, opts, args[0])
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "%s", err)
			os.Exit(1)
		}
		return printSchedules(*getOutputFormat, []*workflow.ScheduleOutputWide{ws})
	},
}

func init() {
	getOutputFormat = workflowcmd.OutputFunc(GetCmd)
	ScheduleCmd.AddCommand(GetCmd)
}
