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
	"runtime"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
)

var stopAppID string

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
			if runtime.GOOS == string(WINDOWS) {
				print.FailureStatusEvent(os.Stderr, "The stop command with run file is not supported on Windows")
				os.Exit(1)
			}
			runFilePath, err = getRunFilePath(runFilePath)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "Failed to get run file path: %v", err)
				os.Exit(1)
			}
			err = executeStopWithRunFile(runFilePath)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, "Failed to stop Dapr and app processes: %s", err)
			} else {
				print.SuccessStatusEvent(os.Stdout, "Dapr and app processes stopped successfully")
			}
			return
		}
		if stopAppID != "" {
			args = append(args, stopAppID)
		}
		cliPIDToNoOfApps := standalone.GetCLiPIDCountMap()
		for _, appID := range args {
			err = standalone.Stop(appID, cliPIDToNoOfApps)
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
	StopCmd.Flags().StringVarP(&runFilePath, "run-file", "f", "", "Path to the configuration file for the apps to run")
	StopCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(StopCmd)
}

func executeStopWithRunFile(runFilePath string) error {
	absFilePath, err := filepath.Abs(runFilePath)
	if err != nil {
		return fmt.Errorf("failed to get abosulte file path for %s: %w", runFilePath, err)
	}
	return standalone.StopAppsWithRunFile(absFilePath)
}
