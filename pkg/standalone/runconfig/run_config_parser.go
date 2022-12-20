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

package runconfig

import (
	"fmt"
	"os"

	"github.com/dapr/cli/utils"
	"gopkg.in/yaml.v2"
)

func (a *AppsRunConfig) ParseAppsConfig(configFile string) error {
	bytes, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(bytes, &a)
	if err != nil {
		return err
	}
	return nil
}

func (a *AppsRunConfig) ValidateRunConfig() error {
	if a.Version == 0 {
		return fmt.Errorf("required filed %q not found in the provided app config file", "version")
	}
	// validate all paths in commons.
	err := utils.ValidateFilePaths(a.Common.ConfigFile, a.Common.ResourcesPath)
	if err != nil {
		return err
	}
	for _, app := range a.Apps {
		if app.AppDir == "" {
			return fmt.Errorf("required filed %q not found in the provided app config file", "app_dir")
		}
		// validate all paths in apps.
		err := utils.ValidateFilePaths(app.ConfigFile, app.ResourcesPath, app.AppDir)
		if err != nil {
			return err
		}
	}
	return nil
}
