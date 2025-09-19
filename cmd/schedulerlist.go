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
	"os"
	"slices"

	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/scheduler"
	"github.com/dapr/cli/utils"
	"github.com/dapr/kit/ptr"
	"github.com/dapr/kit/signals"
)

const (
	schedulerListOutputFormatShort = "short"
	schedulerListOutputFormatWide  = "wide"
	schedulerListOutputFormatYAML  = "yaml"
	schedulerListOutputFormatJSON  = "json"
)

var (
	schedulerListFilterType   string
	schedulerListOutputFormat string
)

var SchedulerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled jobs in the Scheduler",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		if !slices.Contains([]string{
			scheduler.FilterJobsAll,
			scheduler.FilterJobsJob,
			scheduler.FilterJobsActor,
		}, schedulerListFilterType) {
			return errors.New("invalid value for --filter-type. Supported values are 'all', 'jobs', 'actorreminder'.")
		}

		if !slices.Contains([]string{
			schedulerListOutputFormatShort,
			schedulerListOutputFormatWide,
			schedulerListOutputFormatYAML,
			schedulerListOutputFormatJSON,
		}, schedulerListOutputFormat) {
			return errors.New("invalid value for --output. Supported values are 'table', 'wide', 'yaml', 'json'.")
		}

		opts := scheduler.ListJobsOptions{
			SchedulerNamespace: schedulerSchedulerNamespace,
			KubernetesMode:     kubernetesMode,
			FilterJobType:      schedulerListFilterType,
		}
		if schedulerNamespace != "" {
			opts.DaprNamespace = ptr.Of(schedulerNamespace)
		}

		var list any
		var err error
		if schedulerListOutputFormat == schedulerListOutputFormatShort {
			list, err = scheduler.ListJobsAsOutput(ctx, opts)
		} else {
			list, err = scheduler.ListJobsAsOutputWide(ctx, opts)
		}
		if err != nil {
			return err
		}

		switch schedulerListOutputFormat {
		case schedulerListOutputFormatYAML:
			err = utils.PrintDetail(os.Stdout, "yaml", list)
		case schedulerListOutputFormatJSON:
			err = utils.PrintDetail(os.Stdout, "json", list)
		default:
			table, err := gocsv.MarshalString(list)
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
	SchedulerListCmd.Flags().StringVar(&schedulerListFilterType, "filter-type", scheduler.FilterJobsAll, "Filter jobs by type. Supported values are 'all', 'jobs', 'actorreminder'")
	SchedulerListCmd.Flags().StringVarP(&schedulerListOutputFormat, "output", "o", schedulerListOutputFormatShort, "Output format. One of 'short', 'wide', 'yaml', 'json'")
	SchedulerCmd.AddCommand(SchedulerListCmd)
}
