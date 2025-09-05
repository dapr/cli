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
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/workflow"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	workflowAppID      string
	workflowInstanceID string
	workflowNamespace  string
)

var WorkflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Workflow management commands",
}

var WorkflowGetHistoryCmd = &cobra.Command{
	Use:   "get-history",
	Short: "Get workflow history for an app instance (self-hosted)",
	Run: func(cmd *cobra.Command, args []string) {
		if workflowAppID == "" || workflowInstanceID == "" {
			print.FailureStatusEvent(os.Stderr, "--app-id and --instance-id are required")
			os.Exit(1)
		}

		ctx := context.Background()
		events, err := workflow.FetchHistory(ctx, workflowAppID, workflowNamespace, workflowInstanceID)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "%s", err)
			os.Exit(1)
		}

		if len(events) == 0 {
			fmt.Println("No history events found.")
			return
		}

		marshaler := &protojson.MarshalOptions{Multiline: true, Indent: "  "}
		for _, e := range events {
			b, err := marshaler.Marshal(e)
			if err != nil {
				fmt.Println(e)
				continue
			}
			fmt.Println(string(b))
		}
	},
}

func init() {
	WorkflowGetHistoryCmd.Flags().StringVarP(&workflowAppID, "app-id", "a", "", "The application id")
	WorkflowGetHistoryCmd.Flags().StringVarP(&workflowInstanceID, "instance-id", "i", "", "The workflow instance id")
	WorkflowGetHistoryCmd.Flags().StringVarP(&workflowNamespace, "namespace", "n", "default", "The namespace where the workflow app is running")
	WorkflowGetHistoryCmd.MarkFlagRequired("app-id")
	WorkflowGetHistoryCmd.MarkFlagRequired("instance-id")

	WorkflowCmd.AddCommand(WorkflowGetHistoryCmd)
	RootCmd.AddCommand(WorkflowCmd)
}
