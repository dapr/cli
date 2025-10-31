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
	"github.com/dapr/kit/ptr"
	"github.com/dapr/kit/signals"
)

var (
	flagReRunEventID       uint32
	flagReRunNewInstanceID string
	flagReRunInput         *inputFlag
)

var ReRunCmd = &cobra.Command{
	Use:   "rerun [instance ID]",
	Short: "ReRun a workflow instance from the beginning or a specific event. Optionally, a new instance ID and input to the starting event can be provided.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		opts := workflow.ReRunOptions{
			KubernetesMode: flagKubernetesMode,
			Namespace:      flagDaprNamespace,
			AppID:          appID,
			InstanceID:     args[0],
			Input:          flagReRunInput.input,
			EventID:        flagReRunEventID,
		}

		if cmd.Flags().Changed("new-instance-id") {
			opts.NewInstanceID = ptr.Of(flagReRunNewInstanceID)
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
	flagReRunInput = inputCmd(ReRunCmd)
	ReRunCmd.Flags().StringVar(&flagReRunNewInstanceID, "new-instance-id", "", "Optional new ID for the re-run workflow instance. If not provided, a new ID will be generated.")
	ReRunCmd.Flags().Uint32VarP(&flagReRunEventID, "event-id", "e", 0, "The event ID from which to re-run the workflow. If not provided, the workflow will re-run from the beginning.")

	WorkflowCmd.AddCommand(ReRunCmd)
}
