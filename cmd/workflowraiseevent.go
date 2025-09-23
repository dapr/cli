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
	"strings"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/kit/ptr"
	"github.com/dapr/kit/signals"
)

var (
	workflowRaiseEventInput string
)

var WorkflowRaiseEventCmd = &cobra.Command{
	Use:   "raise-event",
	Short: "Raise an event for a workflow waiting for an external event. Expects a single argument '<instance-id>/<event-name>'.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		if len(args) != 1 {
			return errors.New("a single argument, the instance ID and event name separated by a '/', is required")
		}

		split := strings.Split(args[0], "/")
		if len(split) != 2 {
			return errors.New("the argument must be in the format '<instance-id>/<event-name>'")
		}
		instanceID := split[0]
		eventName := split[1]

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		opts := workflow.RaiseEventOptions{
			KubernetesMode: kubernetesMode,
			Namespace:      workflowNamespace,
			AppID:          appID,
			InstanceID:     instanceID,
			Name:           eventName,
		}

		if cmd.Flags().Changed("input") {
			opts.Input = ptr.Of(workflowRaiseEventInput)
		}

		err = workflow.RaiseEvent(ctx, opts)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, err.Error())
			os.Exit(1)
		}

		print.InfoStatusEvent(os.Stdout, "Workflow '%s' raised event '%s' successfully", instanceID, eventName)

		return nil
	},
}

func init() {
	WorkflowRaiseEventCmd.Flags().StringVarP(&workflowRaiseEventInput, "input", "", "", "Optional input data for the event in JSON string format.")

	WorkflowCmd.AddCommand(WorkflowRaiseEventCmd)
}
