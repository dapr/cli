/*
Co
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
	"github.com/dapr/kit/signals"
)

var (
	workflowTerminateOutput string
)

var WorkflowTerminateCmd = &cobra.Command{
	Use:   "terminate",
	Short: "Terminate a workflow instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		if len(args) != 1 {
			return errors.New("a single argument, the instance ID is required")
		}

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		var output *string
		if cmd.Flags().Changed("output") {
			output = &workflowTerminateOutput
		}

		opts := workflow.TerminateOptions{
			KubernetesMode: kubernetesMode,
			Namespace:      workflowNamespace,
			AppID:          appID,
			InstanceID:     args[0],
			Output:         output,
		}

		err = workflow.Terminate(ctx, opts)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}

		print.InfoStatusEvent(os.Stdout, "Workflow '%s' terminated successfully", args[0])

		return nil
	},
}

func init() {
	WorkflowTerminateCmd.Flags().StringVarP(&workflowTerminateOutput, "output", "o", "", "Optional output data for the workflow in JSON string format")
	WorkflowCmd.AddCommand(WorkflowTerminateCmd)
}
