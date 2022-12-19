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

func (a *AppsRunConfig) ParseAppsConfig(configFile string) {
	bytes, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(bytes, &a)
	if err != nil {
		panic(err)
	}
}

func (a *AppsRunConfig) ValidateRunConfig() error {
	if a.Version == 0 {
		return fmt.Errorf("version is required")
	}
	// validate all paths in commons.
	err := utils.ValidateFilePaths(a.Common.ConfigFile, a.Common.ResourcesPath)
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
		// validate all paths in apps.
		err := utils.ValidateFilePaths(app.ConfigFile, app.ResourcesPath, app.AppDir)
		if err != nil {
			return err
		}
	}
	return nil
}
