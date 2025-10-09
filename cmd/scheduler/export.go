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

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/scheduler"
	"github.com/dapr/kit/signals"
)

var (
	schedulerExportFile string
)

var SchedulerExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export all jobs and actor reminders to a binary file, including the tracked count.",
	Long: `Export jobs and actor reminders which are scheduled in Scheduler.
Can later be imported using 'dapr scheduler import'.
dapr scheduler export -o output.bin
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		err := scheduler.Export(ctx, scheduler.ExportImportOptions{
			SchedulerNamespace: schedulerNamespace,
			KubernetesMode:     kubernetesMode,
			TargetFile:         schedulerExportFile,
		})
		if err != nil {
			return err
		}

		print.InfoStatusEvent(os.Stdout, "Export to '%s' complete.", schedulerExportFile)

		return nil
	},
}

func init() {
	SchedulerExportCmd.Flags().MarkHidden("namespace")
	SchedulerExportCmd.Flags().StringVarP(&schedulerExportFile, "output-file", "o", "", "Output binary file to export jobs and actor reminders to.")
	SchedulerExportCmd.MarkFlagRequired("output-file")
	SchedulerExportCmd.MarkFlagFilename("output-file")
	SchedulerCmd.AddCommand(SchedulerExportCmd)
}
