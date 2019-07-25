package cmd

import (
	"fmt"
	"os"

	"github.com/actionscore/cli/pkg/api"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "actions",
	Short: "Actions CLI",
	Long: `
   ___  ___________________  _  ______
  / _ |/ ___/_  __/  _/ __ \/ |/ / __/
 / __ / /__  / / _/ // /_/ /    /\ \  
/_/ |_\___/ /_/ /___/\____/_/|_/___/  								
======================================================
A serverless runtime for hyperscale, distributed systems`,
}

// Execute adds all child commands to the root command
func Execute(version, apiVersion string) {
	RootCmd.Version = version
	api.RuntimeAPIVersion = apiVersion

	fmt.Println(api.RuntimeAPIVersion)

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
