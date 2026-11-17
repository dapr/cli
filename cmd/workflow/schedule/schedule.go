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

	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"

	workflowcmd "github.com/dapr/cli/cmd/workflow"
	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/cli/utils"
)

var ScheduleCmd = &cobra.Command{
	Use:     "schedule",
	Aliases: []string{"schedules", "sched"},
	Short:   "Manage workflow schedules, which start a workflow on a cron schedule.",
	Long: `Manage workflow schedules, which start a workflow on a cron schedule.

Each run of a schedule is a new workflow instance with the ID "<schedule name>_<owner>_<time>",
so "dapr workflow list --instance-id-prefix <schedule name>_" lists them.`,
}

func scheduleOptions(cmd *cobra.Command) (workflow.ScheduleOptions, error) {
	kubernetes, namespace, appID, err := workflowcmd.Target(cmd)
	return workflow.ScheduleOptions{
		KubernetesMode: kubernetes,
		Namespace:      namespace,
		AppID:          appID,
	}, err
}

func printSchedules(outputFormat string, wide []*workflow.ScheduleOutputWide) error {
	switch outputFormat {
	case "yaml", "json":
		return utils.PrintDetail(os.Stdout, outputFormat, wide)
	}

	var list any = wide
	if outputFormat == "short" {
		list = workflow.ShortSchedules(wide)
	}
	table, err := gocsv.MarshalString(list)
	if err != nil {
		return err
	}
	utils.PrintTable(table)
	return nil
}
