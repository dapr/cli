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

	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/cli/utils"
	"github.com/dapr/kit/ptr"
	"github.com/dapr/kit/signals"
)

const (
	workflowHistoryOutputFormatShort = "short"
	workflowHistoryOutputFormatWide  = "wide"
	workflowHistoryOutputFormatYAML  = "yaml"
	workflowHistoryOutputFormatJSON  = "json"
)

var (
	workflowHistoryOutputFormat     string
	workflowHistoryConnectionString string
	workflowHistorySQLTable         string
)

var WorkflowHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Get the history of a workflow instance.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		if len(args) != 1 {
			return errors.New("expect a single argument of the workflow instance ID to get")
		}

		if !slices.Contains([]string{
			workflowHistoryOutputFormatShort,
			workflowHistoryOutputFormatWide,
			workflowHistoryOutputFormatYAML,
			workflowHistoryOutputFormatJSON,
		}, workflowHistoryOutputFormat) {
			return errors.New("invalid value for --output. Supported values are 'table', 'wide', 'yaml', 'json'.")
		}

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		opts := workflow.HistoryOptions{
			KubernetesMode: kubernetesMode,
			Namespace:      workflowNamespace,
			AppID:          appID,
			InstanceID:     args[0],
		}

		if cmd.Flags().Changed("connection-string") {
			opts.ConnectionString = ptr.Of(workflowHistoryConnectionString)
		}
		if cmd.Flags().Changed("sql-table-name") {
			opts.SQLTableName = ptr.Of(workflowHistorySQLTable)
		}

		var list any
		if workflowHistoryOutputFormat == workflowHistoryOutputFormatShort {
			list, err = workflow.HistoryShort(ctx, opts)
		} else {
			list, err = workflow.HistoryWide(ctx, opts)
		}
		if err != nil {
			return err
		}
		switch workflowHistoryOutputFormat {
		case workflowHistoryOutputFormatYAML:
			err = utils.PrintDetail(os.Stdout, "yaml", list)
		case workflowHistoryOutputFormatJSON:
			err = utils.PrintDetail(os.Stdout, "json", list)
		default:
			table, err := gocsv.MarshalString(list)
			if err != nil {
				break
			}

			utils.PrintTable(table)
		}

		return nil
	},
}

func init() {
	WorkflowHistoryCmd.Flags().StringVarP(&workflowHistoryOutputFormat, "output", "o", workflowHistoryOutputFormatShort, "Output format. One of 'short', 'wide', 'yaml', 'json'")
	WorkflowHistoryCmd.Flags().StringVarP(&workflowHistoryConnectionString, "connection-string", "c", workflowHistoryConnectionString, "The connection string used to connect and authenticate to the actor state store")
	WorkflowHistoryCmd.Flags().StringVarP(&workflowHistorySQLTable, "sql-table-name", "t", workflowHistorySQLTable, "The name of the table which is used as the actor state store")

	WorkflowCmd.AddCommand(WorkflowHistoryCmd)
}
