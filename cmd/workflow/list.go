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

	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/cli/utils"
	"github.com/dapr/kit/signals"
)

var (
	listFilter       *workflow.Filter
	listOutputFormat *string

	listConn *connFlag
)

var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List workflows for the given app ID.",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		opts := workflow.ListOptions{
			KubernetesMode:   flagKubernetesMode,
			Namespace:        flagDaprNamespace,
			AppID:            appID,
			ConnectionString: listConn.connectionString,
			TableName:        listConn.tableName,
			Filter:           *listFilter,
		}

		var list any
		var empty bool

		switch *listOutputFormat {
		case outputFormatShort:
			ll, err := workflow.ListShort(ctx, opts)
			if err != nil {
				return err
			}
			empty = len(ll) == 0
			list = ll

		default:
			ll, err := workflow.ListWide(ctx, opts)
			if err != nil {
				return err
			}
			empty = len(ll) == 0
			list = ll
		}

		if empty {
			print.FailureStatusEvent(os.Stderr, "No workflow found in namespace %q for app ID %q", flagDaprNamespace, appID)
			return nil
		}

		switch *listOutputFormat {
		case outputFormatYAML:
			err = utils.PrintDetail(os.Stdout, "yaml", list)
		case outputFormatJSON:
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
	listFilter = filterCmd(ListCmd)
	listOutputFormat = outputFunc(ListCmd)
	listConn = connectionCmd(ListCmd)
	WorkflowCmd.AddCommand(ListCmd)
}
