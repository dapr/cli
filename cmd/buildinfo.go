// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"

	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

//nolint
var BuildInfoCmd = &cobra.Command{
	Use:   "build-info",
	Short: "Print build info of Dapr CLI and runtime",
	Example: `
# Print build info
dapr build-info
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(standalone.GetBuildInfo(RootCmd.Version)) //nolint
	},
}

func init() {
	BuildInfoCmd.Flags().BoolP("help", "h", false, "Print this help message")
	RootCmd.AddCommand(BuildInfoCmd)
}
