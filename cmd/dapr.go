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
}

type daprVersion struct {
	CliVersion     string `json:"Cli version"`
	RuntimeVersion string `json:"Runtime version"`
}

var (
	daprVer   daprVersion
	logAsJSON bool
)

// Execute adds all child commands to the root command.
func Execute(version, apiVersion string) {
	RootCmd.Version = version
	api.RuntimeAPIVersion = apiVersion

	daprVer = daprVersion{
		CliVersion:     version,
		RuntimeVersion: strings.ReplaceAll(standalone.GetRuntimeVersion(), "\n", ""),
	}

	cobra.OnInitialize(initConfig)

	setVersion()

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func setVersion() {
	template := fmt.Sprintf("CLI version: %s \nRuntime version: %s", daprVer.CliVersion, daprVer.RuntimeVersion)
	RootCmd.SetVersionTemplate(template)
}

func initConfig() {
	if logAsJSON {
		print.EnableJSONFormat()
	}

	viper.SetEnvPrefix("dapr")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&logAsJSON, "log-as-json", "", false, "Log output in JSON format")
}
