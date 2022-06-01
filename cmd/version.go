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
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
)

var output string

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Dapr runtime and Cli version.",
	Example: `
# Version for Dapr
dapr version --output json
`,
	Run: func(cmd *cobra.Command, args []string) {
		if output != "" && output != "json" {
			print.FailureStatusEvent(os.Stdout, "An invalid output format was specified.")
			os.Exit(1)
		}
		switch output {
		case "":
			// normal output.
			fmt.Printf("CLI version: %s \nRuntime version: %s", daprVer.CliVersion, daprVer.RuntimeVersion)
		case "json":
			// json output.
			b, err := json.Marshal(daprVer)
			if err != nil {
				print.FailureStatusEvent(os.Stderr, err.Error())
				os.Exit(1)
			}
			fmt.Printf("%s", string(b))
		default:
			// fail and exit.
			os.Exit(1)
		}
	},
}

func init() {
	VersionCmd.Flags().BoolP("help", "h", false, "Print this help message")
	VersionCmd.Flags().StringVarP(&output, "output", "o", "", "The output format of the version command. Valid values are: json.")
	RootCmd.AddCommand(VersionCmd)
}
