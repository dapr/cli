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
	"github.com/dapr/kit/ptr"
	"github.com/dapr/kit/signals"
)

var (
	flagWorkflow           string
	flagSchedule           string
	flagInput              string
	flagDueTime            string
	flagExpiration         string
	flagRepeats            uint32
	flagTargetAppID        string
	flagTargetAppNamespace string
	flagOverwrite          bool
	flagSkipMissed         bool
)

var CreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"add"},
	Short:   "Create a workflow schedule.",
	Long: `Create a schedule that starts a workflow on a cron schedule. Accepts a single argument, the schedule name,
of up to 30 letters, digits or '-'.

The schedule uses the syntax of the Dapr Jobs API: a 6-field cron expression with seconds ("0 30 9 * * MON"),
"@every <duration>", or one of @yearly, @monthly, @weekly, @daily and @hourly. A "CRON_TZ=<zone> " prefix sets
the time zone of a cron expression, which is UTC otherwise.

By default a schedule never misses a time: once the app is back after downtime, every time it missed
starts its own run, one after another, as Dapr jobs and actor reminders do. --skip-missed starts one
run for the missed times instead and carries on from the next time after now.`,
	Example: `  dapr workflow schedule create nightly-report --workflow GenerateReport --schedule "0 0 2 * * *"
  dapr workflow schedule create poll --workflow Poll --schedule "@every 5m" --input '{"source":"a"}' --repeats 10`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		sopts, err := scheduleOptions(cmd)
		if err != nil {
			return err
		}
		opts := workflow.CreateScheduleOptions{
			ScheduleOptions: sopts,
			Name:            args[0],
			Workflow:        flagWorkflow,
			Schedule:        flagSchedule,
			Overwrite:       flagOverwrite,
			SkipMissed:      flagSkipMissed,
		}
		if cmd.Flags().Changed("input") {
			opts.Input = &flagInput
		}
		if cmd.Flags().Changed("due-time") {
			if opts.DueTime, err = workflowcmd.ParseDurationTimestamp(flagDueTime, false); err != nil {
				return err
			}
		}
		if cmd.Flags().Changed("expiration") {
			if opts.ExpirationTime, err = workflowcmd.ParseDurationTimestamp(flagExpiration, false); err != nil {
				return err
			}
		}
		if cmd.Flags().Changed("repeats") {
			opts.Repeats = ptr.Of(flagRepeats)
		}
		if cmd.Flags().Changed("target-app-id") {
			opts.TargetAppID = ptr.Of(flagTargetAppID)
		}
		if cmd.Flags().Changed("target-app-namespace") {
			opts.TargetAppNamespace = ptr.Of(flagTargetAppNamespace)
		}

		if err := workflow.CreateSchedule(ctx, opts); err != nil {
			print.FailureStatusEvent(os.Stderr, "%s", err)
			os.Exit(1)
		}

		print.SuccessStatusEvent(os.Stdout, "Workflow schedule %q created", args[0])
		return nil
	},
}

func init() {
	CreateCmd.Flags().StringVarP(&flagWorkflow, "workflow", "w", "", "The name of the workflow each run starts")
	CreateCmd.Flags().StringVarP(&flagSchedule, "schedule", "s", "", `When runs start, for example "0 0 2 * * *" or "@every 1h"`)
	CreateCmd.Flags().StringVarP(&flagInput, "input", "x", "", "Optional input every run is given. Accepts a JSON string.")
	CreateCmd.Flags().StringVar(&flagDueTime, "due-time", "", "Optional time of the first run, in RFC3339 or Go duration format")
	CreateCmd.Flags().StringVar(&flagExpiration, "expiration", "", "Optional time after which no run starts, in RFC3339 or Go duration format")
	CreateCmd.Flags().Uint32Var(&flagRepeats, "repeats", 0, "Optional number of runs after which the schedule completes")
	CreateCmd.Flags().StringVar(&flagTargetAppID, "target-app-id", "", "Optional app ID hosting the workflow each run starts, if not the app owning the schedule")
	CreateCmd.Flags().StringVar(&flagTargetAppNamespace, "target-app-namespace", "", "Optional namespace of --target-app-id")
	CreateCmd.Flags().BoolVar(&flagOverwrite, "overwrite", false, "Replace an existing schedule of the same name")
	CreateCmd.Flags().BoolVar(&flagSkipMissed, "skip-missed", false, "Start one run for the times missed while the app was down, rather than one run for each")
	CreateCmd.MarkFlagRequired("workflow")
	CreateCmd.MarkFlagRequired("schedule")
	ScheduleCmd.AddCommand(CreateCmd)
}
