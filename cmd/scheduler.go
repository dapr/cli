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
	"os"
	"slices"

	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/scheduler"
	"github.com/dapr/cli/utils"
	"github.com/dapr/kit/ptr"
	"github.com/dapr/kit/signals"
)

const (
	schedulerListJobsOutputFormatShort = "short"
	schedulerListJobsOutputFormatWide  = "wide"
	schedulerListJobsOutputFormatYAML  = "yaml"
	schedulerListJobsOutputFormatJSON  = "json"
)

var (
	schedulerSchedulerNamespace   string
	schedulerNamespace            string
	schedulerListJobsFilterType   string
	schedulerListJobsOutputFormat string
)

var SchedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Scheduler management commands",
}

var SchedulerListJobsCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled jobs in the Scheduler",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := signals.Context()

		if !slices.Contains([]string{
			scheduler.FilterJobsAll,
			scheduler.FilterJobsJob,
			scheduler.FilterJobsActor,
			scheduler.FilterJobsWorkflow,
		}, schedulerListJobsFilterType) {
			print.FailureStatusEvent(os.Stderr, "invalid value for --filter-type. Supported values are 'all', 'jobs', 'actorreminder', 'workflow'")
			os.Exit(1)
		}

		if !slices.Contains([]string{
			schedulerListJobsOutputFormatShort,
			schedulerListJobsOutputFormatWide,
			schedulerListJobsOutputFormatYAML,
			schedulerListJobsOutputFormatJSON,
		}, schedulerListJobsOutputFormat) {
			print.FailureStatusEvent(os.Stderr, "invalid value for --output. Supported values are 'table', 'wide', 'yaml', 'json'")
			os.Exit(1)
		}

		opts := scheduler.ListJobsOptions{
			SchedulerNamespace: schedulerSchedulerNamespace,
			KubernetesMode:     kubernetesMode,
			FilterJobType:      schedulerListJobsFilterType,
		}
		if schedulerNamespace != "" {
			opts.DaprNamespace = ptr.Of(schedulerNamespace)
		}

		var list any
		var err error
		if schedulerListJobsOutputFormat == schedulerListJobsOutputFormatShort {
			list, err = scheduler.ListJobsAsOutput(ctx, opts)
		} else {
			list, err = scheduler.ListJobsAsOutputWide(ctx, opts)
		}
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "%s", err)
			os.Exit(1)
		}

		switch schedulerListJobsOutputFormat {
		case schedulerListJobsOutputFormatYAML:
			err = utils.PrintDetail(os.Stdout, "yaml", list)
		case schedulerListJobsOutputFormatJSON:
			err = utils.PrintDetail(os.Stdout, "json", list)
		default:
			table, err := gocsv.MarshalString(list)
			if err != nil {
				break
			}

			utils.PrintTable(table)
		}

		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	SchedulerListJobsCmd.Flags().StringVar(&schedulerListJobsFilterType, "filter-type", scheduler.FilterJobsAll, "Filter jobs by type. Supported values are 'all', 'jobs', 'actorreminder', 'workflow'")
	SchedulerListJobsCmd.Flags().StringVarP(&schedulerListJobsOutputFormat, "output", "o", schedulerListJobsOutputFormatShort, "Output format. One of 'short', 'wide', 'yaml', 'json'")
	SchedulerListJobsCmd.Flags().StringVar(&schedulerSchedulerNamespace, "scheduler-namespace", "dapr-system", "Kubernetes namespace where the scheduler is deployed")
	SchedulerListJobsCmd.Flags().StringVarP(&schedulerNamespace, "namespace", "n", "", "Kubernetes namespace to list Dapr apps from. If not specified, uses all namespaces")
	SchedulerListJobsCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "List all Dapr pods in a Kubernetes cluster")
	SchedulerCmd.AddCommand(SchedulerListJobsCmd)
	RootCmd.AddCommand(SchedulerCmd)
}
