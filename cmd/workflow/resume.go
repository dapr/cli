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

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/kit/signals"
	"github.com/spf13/cobra"
)

var (
	flagResumeReason string
)

var ResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a workflow that is suspended.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		opts := workflow.ResumeOptions{
			KubernetesMode: flagKubernetesMode,
			Namespace:      flagDaprNamespace,
			AppID:          appID,
			InstanceID:     args[0],
			Reason:         flagResumeReason,
		}

		if err = workflow.Resume(ctx, opts); err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}

		print.InfoStatusEvent(os.Stdout, "Workflow '%s' resumed successfully", args[0])

		return nil
	},
}

func init() {
	ResumeCmd.Flags().StringVarP(&flagResumeReason, "reason", "r", "", "Reason for resuming the workflow")

	WorkflowCmd.AddCommand(ResumeCmd)
}
