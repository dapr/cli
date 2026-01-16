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

package workflow

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/kit/signals"
)

var (
	flagRunInstanceID *instanceIDFlag
	flagRunInput      *inputFlag
	flagRunStartTime  string
)

var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a workflow instance.",
	Long:  "Run a workflow instance based on a given workflow name. Accepts a single argument, the workflow name.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		opts := workflow.RunOptions{
			KubernetesMode: flagKubernetesMode,
			Namespace:      flagDaprNamespace,
			AppID:          appID,
			Name:           args[0],
			InstanceID:     flagRunInstanceID.instanceID,
			Input:          flagRunInput.input,
		}

		if cmd.Flags().Changed("start-time") {
			opts.StartTime, err = parseWorkflowDurationTimestamp(flagRunStartTime, false)
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
	flagRunInstanceID = instanceIDCmd(RunCmd)
	flagRunInput = inputCmd(RunCmd)
	RunCmd.Flags().StringVarP(&flagRunStartTime, "start-time", "s", "", "Optional start time for the workflow in RFC3339 or Go duration string format. If not provided, the workflow starts immediately. A duration of '0s', or any start time, will cause the command to not wait for the command to start")
	WorkflowCmd.AddCommand(RunCmd)
}
