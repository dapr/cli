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
	workflowRunInstanceID string
	workflowRunInput      string
	workflowRunStartTime  string
)

var WorkflowRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a workflow instance based on a given workflow name. Accepts a single argument, the workflow name.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		if len(args) != 1 {
			return errors.New("a single argument, the workflow name, is required")
		}

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		opts := workflow.RunOptions{
			KubernetesMode: kubernetesMode,
			Namespace:      workflowNamespace,
			AppID:          appID,
			Name:           args[0],
		}

		if cmd.Flags().Changed("instance-id") {
			opts.InstanceID = ptr.Of(workflowRunInstanceID)
		}
		if cmd.Flags().Changed("input") {
			opts.Input = ptr.Of(workflowRunInput)
		}
		if cmd.Flags().Changed("start-time") {
			opts.StartTime, err = parseWorkflowDurationTimestamp(workflowRunStartTime, false)
			if err != nil {
				return err
			}
		}

		id, err := workflow.Run(ctx, opts)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}

		print.InfoStatusEvent(os.Stdout, "Workflow instance started successfully: %s", id)

		return nil
	},
}

func init() {
	WorkflowRunCmd.Flags().StringVarP(&workflowRunInstanceID, "instance-id", "i", "", "Optional instance ID for the workflow. If not provided, a random UID will be generated.")
	WorkflowRunCmd.Flags().StringVarP(&workflowRunInput, "input", "", "", "Optional input data for the workflow in JSON string format.")
	WorkflowRunCmd.Flags().StringVarP(&workflowRunStartTime, "start-time", "s", "", "Optional start time for the workflow in RFC3339 or Go duration string format. If not provided, the workflow starts immediately. A duration of '0s', or any start time, will cause the command to not wait for the command to start")

	WorkflowCmd.AddCommand(WorkflowRunCmd)
}
