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

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
)

var BuildInfoCmd = &cobra.Command{
	Use:   "build-info",
	Short: "Print build info of Dapr CLI and runtime",
	Example: `
# Print build info
dapr build-info
`,
	Run: func(cmd *cobra.Command, args []string) {
		out, err := standalone.GetBuildInfo(daprPath, RootCmd.Version)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, "Error getting build info: %s", err.Error())
			os.Exit(1)
		}
		fmt.Println(out)
	},
}

func init() {
	BuildInfoCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(BuildInfoCmd)
}
