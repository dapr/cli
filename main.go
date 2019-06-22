package main

import "github.com/actionscore/cli/cmd"

// Value for version is injected by the build
var (
	version = "edge"
)

func main() {
	cmd.Execute(version)
}
