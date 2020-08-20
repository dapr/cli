// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package main

import (
	"os"

	"github.com/dapr/cli/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	if len(os.Args) < 2 {
		panic("Requires a path to generate docs in.")
	}
	doc.GenMarkdownTree(cmd.RootCmd, os.Args[1])
}
