/*
Copyright 2022 The Dapr Authors
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

package standalone

import (
	"fmt"
	"os"

	"github.com/dapr/cli/utils"
	"gopkg.in/yaml.v2"
)

type AppsRunConfig struct {
	Common  Common `yaml:"common"`
	Apps    []Apps `yaml:"apps"`
	Version int    `yaml:"version"`
}

type Apps struct {
	Common `yaml:",inline"`
	AppDir string `yaml:"app_dir"`
}

type Common struct {
	Env       []EnvItems `yaml:"env"`
	RunConfig `yaml:",inline"`
}

type EnvItems struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func (a *AppsRunConfig) GetApps(configFile string) {
	bytes, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(bytes, &a)
	if err != nil {
		panic(err)
	}
	err = validateRunConfig(a)
	if err != nil {
		panic(err)
	}

	// TODO: remove this later
	printStructFields(a)
}

func validateRunConfig(a *AppsRunConfig) error {
	if a.Version == 0 {
		return fmt.Errorf("version is required")
	}
	// validate all paths in commons
	allCommonPaths := []string{a.Common.ConfigFile, a.Common.ResourcesPath}
	err := validateFilePaths(allCommonPaths)
	if err != nil {
		return err
	}
	for _, app := range a.Apps {
		if app.AppID == "" {
			return fmt.Errorf("app id is required")
		}
		if app.AppDir == "" {
			return fmt.Errorf("app dir is required")
		}
		// validate all paths in apps
		allAppsPaths := []string{app.ConfigFile, app.ResourcesPath, app.AppDir}
		err := validateFilePaths(allAppsPaths)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateFilePaths(filePaths []string) error {
	for _, path := range filePaths {
		if path != "" {
			_, err := utils.IsFilePathValid(path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func printStructFields(a *AppsRunConfig) {
	for _, env := range a.Common.Env {
		fmt.Println("Name:", env.Name, "Value:", env.Value)
	}
	fmt.Println("====================================")
	fmt.Println("Common Resource Dir: ", a.Common.ResourcesPath)
	fmt.Println("Common Config File: ", a.Common.ConfigFile)
	fmt.Println("Common App Port: ", a.Common.AppPort)
	fmt.Println("Common App Protocol: ", a.Common.AppProtocol)
	fmt.Println("Common Unix Domain Socket: ", a.Common.UnixDomainSocket)
	fmt.Println("====================================")
	for _, app := range a.Apps {
		fmt.Println("\nApp ID:", app.AppID)
		fmt.Println("App dir:", app.AppDir)
		fmt.Println("App resources dir:", app.ResourcesPath)
		fmt.Println("App config file:", app.ConfigFile)
		fmt.Println("App protocol:", app.AppProtocol)
		fmt.Println("App port:", app.AppPort)
		fmt.Println("App command:", app.Command)
		fmt.Println("App Unix domain socket:", app.UnixDomainSocket)
		for _, env := range app.Env {
			fmt.Println("app env Name:", env.Name, "app env Value:", env.Value)
		}
	}
}
