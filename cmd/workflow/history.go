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

	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/cli/utils"
	"github.com/dapr/kit/signals"
)

var (
	historyOutputFormat *string
)

var HistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Get the history of a workflow instance.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := signals.Context()

		appID, err := getWorkflowAppID(cmd)
		if err != nil {
			return err
		}

		opts := workflow.HistoryOptions{
			KubernetesMode: flagKubernetesMode,
			Namespace:      flagDaprNamespace,
			AppID:          appID,
			InstanceID:     args[0],
		}

		var list any
		if *historyOutputFormat == outputFormatShort {
			list, err = workflow.HistoryShort(ctx, opts)
		} else {
			list, err = workflow.HistoryWide(ctx, opts)
		}
		if err != nil {
			return err
		}

		switch *historyOutputFormat {
		case outputFormatYAML:
			err = utils.PrintDetail(os.Stdout, "yaml", list)
		case outputFormatJSON:
			err = utils.PrintDetail(os.Stdout, "json", list)
		default:
			var table string
			table, err = gocsv.MarshalString(list)
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
	historyOutputFormat = outputFunc(HistoryCmd)
	WorkflowCmd.AddCommand(HistoryCmd)
}
