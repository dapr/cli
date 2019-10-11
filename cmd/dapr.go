// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/api"
	"github.com/dapr/cli/pkg/version"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "dapr",
	Short: "Dapr CLI",
	Long: `
	 __                
    ____/ /___ _____  _____
   / __  / __ '/ __ \/ ___/
  / /_/ / /_/ / /_/ / /    
  \__,_/\__,_/ .___/_/     
	      /_/            
									   
======================================================
A serverless runtime for hyperscale, distributed systems`,
}

// Execute adds all child commands to the root command
func Execute(version, apiVersion string) {
	RootCmd.Version = version
	api.RuntimeAPIVersion = apiVersion

	setVersion()

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func setVersion() {
	template := fmt.Sprintf("cli version: %s \nruntime version: %s", RootCmd.Version, version.GetRuntimeVersion())
	RootCmd.SetVersionTemplate(template)
}
