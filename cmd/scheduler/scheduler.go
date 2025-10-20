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

package scheduler

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/scheduler"
)

const (
	outputFormatShort = "short"
	outputFormatWide  = "wide"
	outputFormatYAML  = "yaml"
	outputFormatJSON  = "json"
)

var (
	daprNamespace      string
	schedulerNamespace string
	kubernetesMode     bool
)

var SchedulerCmd = &cobra.Command{
	Use:     "scheduler",
	Short:   "Scheduler management commands. Use -k to target a Kubernetes Dapr cluster.",
	Aliases: []string{"sched"},
}

func init() {
	SchedulerCmd.PersistentFlags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Perform scheduler command on a Kubernetes Dapr cluster")
	SchedulerCmd.PersistentFlags().StringVarP(&daprNamespace, "namespace", "n", "default", "Namespace of the Dapr application")
	SchedulerCmd.PersistentFlags().StringVar(&schedulerNamespace, "scheduler-namespace", "dapr-system", "Kubernetes namespace where the scheduler is deployed, only relevant if --kubernetes is set")
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
			return errors.New("invalid value for --output. Supported values are " + strings.Join(outputs, ", "))
		}

		if pre != nil {
			return pre(cmd, args)
		}
		return nil
	}

	return &outputFormat
}

func filterFunc(cmd *cobra.Command) *string {
	all := []string{
		scheduler.FilterAll,
		scheduler.FilterJob,
		scheduler.FilterActor,
		scheduler.FilterWorkflow,
		scheduler.FilterActivity,
	}

	var filterType string
	cmd.Flags().StringVar(&filterType, "filter", scheduler.FilterAll,
		fmt.Sprintf("Filter jobs by type. Supported values are %s\n", strings.Join(all, ", ")),
	)

	pre := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if !slices.Contains(all, filterType) {
			return errors.New("invalid value for --filter. Supported values are " + strings.Join(all, ", "))
		}

		if pre != nil {
			return pre(cmd, args)
		}
		return nil
	}

	return &filterType
}
