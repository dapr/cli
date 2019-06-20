package main

import "github.com/actionscore/cli/cmd"

var (
	VERSION = "0.0.1"
)

func main() {
	cmd.Execute(VERSION)
}
