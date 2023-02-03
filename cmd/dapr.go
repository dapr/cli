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
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dapr/cli/pkg/api"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
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

===============================
Distributed Application Runtime`,
	Run: func(cmd *cobra.Command, args []string) {
		if versionFlag {
			printVersion()
		}
	},
}

type daprVersion struct {
	CliVersion     string `json:"Cli version"`
	RuntimeVersion string `json:"Runtime version"`
}

type osType string

const (
	windowsOsType osType = "windows"
)

var (
	cliVersion  string
	versionFlag bool
	daprVer     daprVersion
	logAsJSON   bool
	daprPath    string
)

// Execute adds all child commands to the root command.
func Execute(version, apiVersion string) {
	// Need to be set here as it is accessed in initConfig.
	cliVersion = version
	api.RuntimeAPIVersion = apiVersion

	cobra.OnInitialize(initConfig)

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func printVersion() {
	template := fmt.Sprintf(cliVersionTemplateString, daprVer.CliVersion, daprVer.RuntimeVersion)
	fmt.Printf(template)
}

// Function is called as a preRun initializer for each command executed.
func initConfig() {
	if logAsJSON {
		print.EnableJSONFormat()
	}
	// err intentionally ignored since daprd may not yet be installed.
	runtimeVer, _ := standalone.GetRuntimeVersion(daprPath)

	daprVer = daprVersion{
		// Set in Execute() method in this file before initConfig() is called by cmd.Execute().
		CliVersion:     cliVersion,
		RuntimeVersion: strings.ReplaceAll(runtimeVer, "\n", ""),
	}

	viper.SetEnvPrefix("dapr")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

func init() {
	RootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "version for dapr")
	RootCmd.PersistentFlags().StringVarP(&daprPath, "dapr-path", "", "", "The path to the dapr installation directory")
	RootCmd.PersistentFlags().BoolVarP(&logAsJSON, "log-as-json", "", false, "Log output in JSON format")
}
