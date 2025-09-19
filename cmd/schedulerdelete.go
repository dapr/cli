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
	"errors"
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/scheduler"
	"github.com/dapr/kit/signals"
	"github.com/spf13/cobra"
)

var (
	schedulerDeleteAll bool
)

var SchedulerDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete jobs or actor reminders which are scheduled in Scheduler.",
	Long: `Delete jobs or actor reminders which are scheduled in Scheduler.
Namespace (-n) is required.
Job names are formatted by their type, app ID, then identifier.
Actor reminders require the actor type, actor ID, then reminder name, separated by ||.
Accepts multiple job names or actor reminders to delete.

dapr scheduler delete -n foo job/my-app-id/my-job-name
dapr scheduler delete -n foo "actorreminder/my-actor-type||my-actor-id||my-reminder-name"
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		if !cmd.Flag("namespace").Changed {
			return errors.New(`required flag(s) "--namespace" not set`)
		}

		if schedulerDeleteAll {
			if err := scheduler.DeleteAll(ctx, scheduler.DeleteOptions{
				SchedulerNamespace: schedulerSchedulerNamespace,
				DaprNamespace:      schedulerNamespace,
				KubernetesMode:     kubernetesMode,
			}); err != nil {
				return fmt.Errorf("Failed to delete jobs: %s", err)
			}
			return nil
		}

		if len(args) < 1 {
			return errors.New(`Qualifier and job name are required.
Example: dapr scheduler delete -n foo job/my-app-id/my-job-name
Example: dapr scheduler delete -n foo "actorreminder/my-actor-type||my-actor-id||my-reminder-name"`)
		}

		for _, name := range args {
			if err := scheduler.Delete(ctx, name, scheduler.DeleteOptions{
				SchedulerNamespace: schedulerSchedulerNamespace,
				DaprNamespace:      schedulerNamespace,
				KubernetesMode:     kubernetesMode,
			}); err != nil {
				return fmt.Errorf("Failed to delete job: %s", err)
			}
			print.InfoStatusEvent(os.Stdout, "Deleted job '%s' in namespace '%s'.", name, schedulerNamespace)
		}

		return nil
	},
}

func init() {
	SchedulerDeleteCmd.Flags().BoolVar(&schedulerDeleteAll, "delete-all-yes-i-know-what-i-am-doing", false, "Deletes all jobs and actor reminders in the given namespace.")
	SchedulerCmd.AddCommand(SchedulerDeleteCmd)
}
