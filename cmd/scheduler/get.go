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
	"os"

	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/scheduler"
	"github.com/dapr/cli/utils"
	"github.com/dapr/kit/signals"
)

var (
	getOutputFormat *string
)

var GetCmd = &cobra.Command{
	Use:     "get",
	Aliases: []string{"g", "ge"},
	Short: `Get a scheduled app job or actor reminder in Scheduler.
Job names are formatted by their type, app ID, then identifier.
Actor reminders require the actor type, actor ID, then reminder name, separated by /.
Workflow reminders require the app ID, instance ID, then reminder name, separated by /.
Activity reminders require the app ID, activity ID, separated by /.
Accepts multiple names.
`,
	Args: cobra.MinimumNArgs(1),
	Example: `
dapr scheduler get app/my-app-id/my-job-name
dapr scheduler get actor/my-actor-type/my-actor-id/my-reminder-name
dapr scheduler get workflow/my-app-id/my-instance-id/my-workflow-reminder-name
dapr scheduler get activity/my-app-id/xyz::0::1
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()
		opts := scheduler.GetOptions{
			SchedulerNamespace: schedulerNamespace,
			KubernetesMode:     kubernetesMode,
			DaprNamespace:      daprNamespace,
		}

		var list any
		var err error
		if *getOutputFormat == outputFormatShort {
			list, err = scheduler.Get(ctx, opts, args...)
		} else {
			list, err = scheduler.GetWide(ctx, opts, args...)
		}
		if err != nil {
			return err
		}

		switch *getOutputFormat {
		case outputFormatYAML:
			err = utils.PrintDetail(os.Stdout, "yaml", list)
		case outputFormatJSON:
			err = utils.PrintDetail(os.Stdout, "json", list)
		default:
			var table string
			table, err = gocsv.MarshalString(list)
			if err != nil {
				break
			}

			utils.PrintTable(table)
		}

		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	getOutputFormat = outputFunc(GetCmd)
	SchedulerCmd.AddCommand(GetCmd)
}
