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

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/kit/ptr"
	"github.com/dapr/kit/signals"
)

var (
	workflowReRunEventID       uint32
	workflowReRunNewInstanceID string
	workflowReRunInput         string
)

var WorkflowReRunCmd = &cobra.Command{
	Use:   "rerun [instance ID]",
	Short: "ReRun a workflow instance from the beginning or a specific event. Optionally, a new instance ID and input to the starting event can be provided.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		if len(args) != 1 {
			return errors.New("instance ID argument is required")
		}

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		opts := workflow.ReRunOptions{
			KubernetesMode: kubernetesMode,
			Namespace:      workflowNamespace,
			AppID:          appID,
			InstanceID:     args[0],
			EventID:        workflowReRunEventID,
		}

		if cmd.Flags().Changed("new-instance-id") {
			opts.NewInstanceID = ptr.Of(workflowReRunNewInstanceID)
		}
		if cmd.Flags().Changed("event-id") {
			opts.Input = ptr.Of(workflowReRunInput)
		}

		id, err := workflow.ReRun(ctx, opts)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}

		print.InfoStatusEvent(os.Stdout, "Rerunning workflow instance: %s", id)

		return nil
	},
}

func init() {
	WorkflowReRunCmd.Flags().StringVar(&workflowReRunNewInstanceID, "new-instance-id", "", "Optional new ID for the re-run workflow instance. If not provided, a new ID will be generated.")
	WorkflowReRunCmd.Flags().Uint32VarP(&workflowReRunEventID, "event-id", "e", 0, "The event ID from which to re-run the workflow. If not provided, the workflow will re-run from the beginning.")
	WorkflowReRunCmd.Flags().StringVar(&workflowReRunInput, "input", "", "Optional input data for the starting event of the re-run workflow instance.")

	WorkflowCmd.AddCommand(WorkflowReRunCmd)
}
