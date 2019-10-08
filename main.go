// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package main

import (
	"github.com/dapr/cli/cmd"
)

// Values for version and apiVersion are injected by the build
var (
	version    = ""
	apiVersion = "1.0"
)

func main() {
	cmd.Execute(version, apiVersion)
}
