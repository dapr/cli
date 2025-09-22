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

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/kit/signals"
)

var (
	workflowPurgeOlderThan string
	workflowPurgeAll       bool
)

var WorkflowPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge one or more workflow instances with a terminal state. Accepts a workflow instance ID argument or flags to purge multiple instances.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		switch {
		case cmd.Flags().Changed("all-older-than"), cmd.Flags().Changed("purge-all-yes-i-know-what-i-am-doing"):
			if len(args) > 0 {
				return errors.New("no arguments are accepted when using purge all flags")
			}
		default:
			if len(args) == 0 {
				return errors.New("one or more workflow instance ID arguments are required when not using purge all flags")
			}
		}

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		opts := workflow.PurgeOptions{
			KubernetesMode: kubernetesMode,
			Namespace:      workflowNamespace,
			AppID:          appID,
			InstanceIDs:    args,
			All:            workflowPurgeAll,
		}

		if cmd.Flags().Changed("all-older-than") {
			opts.AllOlderThan, err = parseWorkflowDurationTimestamp(workflowPurgeOlderThan, true)
			if err != nil {
				return err
			}
		}

		return workflow.Purge(ctx, opts)
	},
}

func init() {
	WorkflowPurgeCmd.Flags().StringVar(&workflowPurgeOlderThan, "all-older-than", "", "Purge workflow instances older than the specified Go duration or timestamp, e.g., '24h' or '2023-01-02T15:04:05Z'.")
	WorkflowPurgeCmd.Flags().BoolVar(&workflowPurgeAll, "purge-all-yes-i-know-what-i-am-doing", false, "Purge all workflow instances in a terminal state. Use with caution.")
	WorkflowPurgeCmd.MarkFlagsMutuallyExclusive("all-older-than", "purge-all-yes-i-know-what-i-am-doing")

	WorkflowCmd.AddCommand(WorkflowPurgeCmd)
}
