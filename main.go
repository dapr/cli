package main

import (
	"github.com/actionscore/cli/cmd"
)

// Value for version is injected by the build
var (
	version = ""
)

func main() {
	cmd.Execute(version)
}
