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
	"slices"
	"strings"

	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/cli/utils"
	"github.com/dapr/kit/ptr"
	"github.com/dapr/kit/signals"
)

const (
	workflowListOutputFormatShort = "short"
	workflowListOutputFormatWide  = "wide"
	workflowListOutputFormatYAML  = "yaml"
	workflowListOutputFormatJSON  = "json"
)

var (
	workflowListOutputFormat     string
	workflowListAppID            string
	workflowListConnectionString string
	workflowListSQLTable         string

	workflowListFilterWorkflow string
	workflowListFilterStatus   string

	workflowListStatuses = []string{
		"RUNNING",
		"COMPLETED",
		"CONTINUED_AS_NEW",
		"FAILED",
		"CANCELED",
		"TERMINATED",
		"PENDING",
		"SUSPENDED",
	}
)

var WorkflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflows and their status for a given app ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		if !slices.Contains([]string{
			workflowListOutputFormatShort,
			workflowListOutputFormatWide,
			workflowListOutputFormatYAML,
			workflowListOutputFormatJSON,
		}, workflowListOutputFormat) {
			return errors.New("invalid value for --output. Supported values are 'table', 'wide', 'yaml', 'json'.")
		}

		opts := workflow.ListOptions{
			KubernetesMode: kubernetesMode,
			Namespace:      workflowNamespace,
			AppID:          workflowListAppID,
		}

		if cmd.Flags().Changed("connection-string") {
			opts.ConnectionString = ptr.Of(workflowListConnectionString)
		}
		if cmd.Flags().Changed("sql-table-name") {
			opts.SQLTableName = ptr.Of(workflowListSQLTable)
		}
		if cmd.Flags().Changed("filter-workflow") {
			opts.FilterWorkflowName = ptr.Of(workflowListFilterWorkflow)
		}
		if cmd.Flags().Changed("filter-status") {
			if !slices.Contains(workflowListStatuses, workflowListFilterStatus) {
				return errors.New("invalid value for --filter-status. Supported values are " + strings.Join(workflowListStatuses, ", "))
			}
			opts.FilterWorkflowStatus = ptr.Of(workflowListFilterStatus)
		}

		var list any
		var err error
		if workflowListOutputFormat == workflowListOutputFormatShort {
			list, err = workflow.ListShort(ctx, opts)
		} else {
			list, err = workflow.ListWide(ctx, opts)
		}
		if err != nil {
			return err
		}
		switch workflowListOutputFormat {
		case workflowListOutputFormatYAML:
			err = utils.PrintDetail(os.Stdout, "yaml", list)
		case workflowListOutputFormatJSON:
			err = utils.PrintDetail(os.Stdout, "json", list)
		default:
			table, err := gocsv.MarshalString(list)
			if err != nil {
				break
			}

			utils.PrintTable(table)
		}

		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	WorkflowListCmd.Flags().StringVarP(&workflowListOutputFormat, "output", "o", workflowListOutputFormatShort, "Output format. One of 'short', 'wide', 'yaml', 'json'")
	WorkflowListCmd.Flags().StringVarP(&workflowListConnectionString, "connection-string", "c", workflowListConnectionString, "The connection string used to connect and authenticate to the actor state store")
	WorkflowListCmd.Flags().StringVarP(&workflowListSQLTable, "sql-table-name", "t", workflowListSQLTable, "The name of the table which is used as the actor state store")
	WorkflowListCmd.Flags().StringVarP(&workflowListAppID, "app-id", "a", "", "The application id")

	WorkflowListCmd.Flags().StringVarP(&workflowListFilterWorkflow, "filter-workflow", "w", "", "List only the workflows with the given name")
	WorkflowListCmd.Flags().StringVarP(&workflowListFilterStatus, "filter-status", "s", "", "List only the workflows with the given runtime status. One of "+strings.Join(workflowListStatuses, ", "))

	WorkflowListCmd.MarkFlagRequired("app-id")
	WorkflowCmd.AddCommand(WorkflowListCmd)
}
