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
	workflowResumeReason string
)

var WorkflowResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a workflow that is suspended",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		if len(args) != 1 {
			return errors.New("a single argument, the instance ID is required")
		}

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		opts := workflow.ResumeOptions{
			KubernetesMode: kubernetesMode,
			Namespace:      workflowNamespace,
			AppID:          appID,
			InstanceID:     args[0],
			Reason:         workflowResumeReason,
		}

		err = workflow.Resume(ctx, opts)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}

		print.InfoStatusEvent(os.Stdout, "Workflow '%s' resumed successfully", args[0])

		return nil
	},
}

func init() {
	WorkflowResumeCmd.Flags().StringVarP(&workflowResumeReason, "reason", "r", "", "Reason for resuming the workflow")
	WorkflowCmd.AddCommand(WorkflowResumeCmd)
}
