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
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/dapr/cli/pkg/workflow"
	"github.com/dapr/kit/ptr"
	kittime "github.com/dapr/kit/time"
	"github.com/spf13/cobra"
)

const (
	outputFormatShort = "short"
	outputFormatWide  = "wide"
	outputFormatYAML  = "yaml"
	outputFormatJSON  = "json"
)

var (
	flagKubernetesMode   bool
	flagDaprNamespace    string
	flagAppID            string
	flagConnectionString string
	flagTable            string
)

var WorkflowCmd = &cobra.Command{
	Use:     "workflow",
	Short:   "Workflow management commands. Use -k to target a Kubernetes Dapr cluster.",
	Aliases: []string{"work"},
}

func init() {
	WorkflowCmd.PersistentFlags().BoolVarP(&flagKubernetesMode, "kubernetes", "k", false, "Target a Kubernetes dapr installation")
	WorkflowCmd.PersistentFlags().StringVarP(&flagDaprNamespace, "namespace", "n", "default", "Namespace to perform workflow operation on")
	WorkflowCmd.PersistentFlags().StringVarP(&flagAppID, "app-id", "a", "", "The app ID owner of the workflow instance")
}

func outputFunc(cmd *cobra.Command) *string {
	outputs := []string{
		outputFormatShort,
		outputFormatWide,
		outputFormatYAML,
		outputFormatJSON,
	}

	var outputFormat string
	cmd.Flags().StringVarP(&outputFormat, "output", "o", outputFormatShort, fmt.Sprintf("Output format. One of %s",
		strings.Join(outputs, ", ")),
	)

	pre := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if !slices.Contains(outputs, outputFormat) {
			return errors.New("invalid value for --output. Supported values are 'table', 'wide', 'yaml', 'json'.")
		}

		if pre != nil {
			return pre(cmd, args)
		}
		return nil
	}

	return &outputFormat
}

func getWorkflowAppID(cmd *cobra.Command) (string, error) {
	if cmd.Flags().Changed("app-id") {
		return flagAppID, nil
	}

	var errRequired = fmt.Errorf("the app ID is required when there are multiple Dapr instances. Please specify it using the --app-id flag")
	var errNotFound = fmt.Errorf("no Dapr instances found. Please ensure that Dapr is running")

	if flagKubernetesMode {
		list, err := kubernetes.List(flagDaprNamespace)
		if err != nil {
			return "", err
		}

		if len(list) == 0 {
			return "", errNotFound
		}

		if len(list) != 1 {
			return "", errRequired
		}

		return list[0].AppID, nil
	}

	list, err := standalone.List()
	if err != nil {
		return "", err
	}

	if len(list) == 0 {
		return "", errNotFound
	}

	if len(list) != 1 {
		return "", errRequired
	}

	return list[0].AppID, nil
}

func parseWorkflowDurationTimestamp(str string, durationPast bool) (*time.Time, error) {
	dur, err := time.ParseDuration(str)
	if err == nil {
		if durationPast {
			dur = -dur
		}
		return ptr.Of(time.Now().Add(dur)), nil
	}

	ts, err := kittime.ParseTime(str, nil)
	if err != nil {
		return nil, err
	}

	return ptr.Of(ts), nil
}

func filterCmd(cmd *cobra.Command) *workflow.Filter {
	filter := new(workflow.Filter)

	var (
		name   string
		status string
		maxAge string

		listStatuses = []string{
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

	cmd.Flags().StringVarP(&name, "filter-name", "w", "", "Filter only the workflows with the given name")
	cmd.Flags().StringVarP(&status, "filter-status", "s", "", "Filter only the workflows with the given runtime status. One of "+strings.Join(listStatuses, ", "))
	cmd.Flags().StringVarP(&maxAge, "filter-max-age", "m", "", "Filter only the workflows started within the given duration or timestamp. Examples: 300ms, 1.5h or 2h45m, 2023-01-02T15:04:05 or 2023-01-02")

	pre := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("filter-name") {
			filter.Name = &name
		}
		if cmd.Flags().Changed("filter-status") {
			if !slices.Contains(listStatuses, status) {
				return errors.New("invalid value for --filter-status. Supported values are " + strings.Join(listStatuses, ", "))
			}
			filter.Status = &status
		}

		if cmd.Flags().Changed("filter-max-age") {
			var err error
			filter.MaxAge, err = parseWorkflowDurationTimestamp(maxAge, true)
			if err != nil {
				return err
			}
		}

		if pre != nil {
			return pre(cmd, args)
		}

		return nil
	}

	return filter
}

type connFlag struct {
	connectionString *string
	tableName        *string
}

func connectionCmd(cmd *cobra.Command) *connFlag {
	var (
		flagConnectionString string
		flagTableName        string
	)

	cmd.Flags().StringVarP(&flagConnectionString, "connection-string", "c", "", "The connection string used to connect and authenticate to the actor state store")
	cmd.Flags().StringVarP(&flagTableName, "table-name", "t", "", "The name of the table or collection which is used as the actor state store")

	var connFlag connFlag
	pre := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("connection-string") {
			connFlag.connectionString = &flagConnectionString
		}

		if cmd.Flags().Changed("table-name") {
			connFlag.tableName = &flagTableName
		}

		if pre != nil {
			return pre(cmd, args)
		}

		return nil
	}

	return &connFlag
}

type instanceIDFlag struct {
	instanceID *string
}

func instanceIDCmd(cmd *cobra.Command) *instanceIDFlag {
	var instanceID string
	iFlag := new(instanceIDFlag)

	cmd.Flags().StringVarP(&instanceID, "instance-id", "i", "", "The target workflow instance ID.")

	pre := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("instance-id") {
			iFlag.instanceID = &instanceID
		}

		if pre != nil {
			return pre(cmd, args)
		}

		return nil
	}

	return iFlag
}

type inputFlag struct {
	input *string
}

func inputCmd(cmd *cobra.Command) *inputFlag {
	var input string
	iFlag := new(inputFlag)

	cmd.Flags().StringVarP(&input, "input", "x", "", "Optional input data for the new workflow instance. Accepts a JSON string.")

	pre := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("input") {
			iFlag.input = &input
		}

		if pre != nil {
			return pre(cmd, args)
		}

		return nil
	}

	return iFlag
}
