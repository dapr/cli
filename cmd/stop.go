/*
Copyright 2021 The Dapr Authors
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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
)

var (
	stopAppID string
	stopK8s   bool
)

var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Dapr instances and their associated apps. Supported platforms: Self-hosted",
	Example: `
# Stop Dapr application
dapr stop --app-id <ID>

# Stop multiple apps by providing a run config file
dapr stop --run-file dapr.yaml

# Stop multiple apps by providing a directory path containing the run config file(dapr.yaml)
dapr stop --run-file /path/to/directory
`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if len(runFilePath) > 0 {
			runFilePath, err = getRunFilePath(runFilePath)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "Failed to get run file path: %v", err)
				os.Exit(1)
			}
			if !stopK8s {
				err = executeStopWithRunFile(runFilePath)
				if err != nil {
					print.FailureStatusEvent(os.Stderr, "Failed to stop Dapr and app processes: %s", err)
				} else {
					print.SuccessStatusEvent(os.Stdout, "Dapr and app processes stopped successfully")
				}
				return
			}
			config, _, cErr := getRunConfigFromRunFile(runFilePath)
			if cErr != nil {
				print.FailureStatusEvent(os.Stderr, "Failed to parse run template file %q: %s", runFilePath, cErr.Error())
			}
			err = kubernetes.Stop(runFilePath, config)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "Error stopping deployments from multi-app run template: %v", err)
			}
		}
		if stopAppID != "" {
			args = append(args, stopAppID)
		}
		apps, err := standalone.List()
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "failed to get list of apps started by dapr : %s", err)
			os.Exit(1)
		}
		cliPIDToNoOfApps := standalone.GetCLIPIDCountMap(apps)
		for _, appID := range args {
			err = standalone.Stop(appID, cliPIDToNoOfApps, apps)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "failed to stop app id %s: %s", appID, err)
			} else {
				print.SuccessStatusEvent(os.Stdout, "app stopped successfully: %s", appID)
			}
		}
	},
}

func init() {
	StopCmd.Flags().StringVarP(&stopAppID, "app-id", "a", "", "The application id to be stopped")
	StopCmd.Flags().StringVarP(&runFilePath, "run-file", "f", "", "Path to the run template file for the list of apps to stop")
	StopCmd.Flags().BoolVarP(&stopK8s, "kubernetes", "k", false, "Stop deployments in Kunernetes based on multi-app run file")
	StopCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(StopCmd)
}

func executeStopWithRunFile(runFilePath string) error {
	absFilePath, err := filepath.Abs(runFilePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute file path for %s: %w", runFilePath, err)
	}
	return standalone.StopAppsWithRunFile(absFilePath)
}
