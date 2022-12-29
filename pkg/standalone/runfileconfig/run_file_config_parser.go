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

package runfileconfig

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/dapr/cli/utils"

	"gopkg.in/yaml.v2"
)

// constants for the keys from the yaml file.
const APPS = "apps"

func (a *RunFileConfig) ParseAppsConfig(runFilePath string) error {
	var err error
	bytes, err := utils.ReadFile(runFilePath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(bytes, &a)
	if err != nil {
		return fmt.Errorf("error in parsing the provided app config file: %w", err)
	}
	return nil
}

// ValidateRunConfig validates the run file config for mandatory fields and valid paths.
func (a *RunFileConfig) ValidateRunConfig(runFilePath string) error {
	baseDir, err := filepath.Abs(filepath.Dir(runFilePath))
	if err != nil {
		return fmt.Errorf("error in getting the absolute path of the provided run template file: %w", err)
	}
	if a.Version == 0 {
		return errors.New("required field 'version' not found in the provided run template file")
	}

	// resolve relative path to absolute and validate all paths in commons.
	err = a.resolvePathToAbsAndValidate(baseDir, &a.Common.ConfigFile, &a.Common.ResourcesPath)
	if err != nil {
		return err
	}
	for i := 0; i < len(a.Apps); i++ {
		if a.Apps[i].AppDirPath == "" {
			return errors.New("required filed 'app_dir_path' not found in the provided app config file")
		}
		// resolve relative path to absolute and validate all paths for app.
		err := a.resolvePathToAbsAndValidate(baseDir, &a.Apps[i].ConfigFile, &a.Apps[i].ResourcesPath, &a.Apps[i].AppDirPath)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetApps returns a list of apps with the merged values forthe keys from common section of the YAML file.
func (a *RunFileConfig) GetApps(filePath string) ([]Apps, error) {
	bytes, err := utils.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	appsKeys, err := a.getAppsKeysFromYaml(bytes)
	if err != nil {
		return nil, err
	}
	if len(a.Apps) != len(appsKeys) {
		return nil, errors.New("error in parsing the provided app config file to extract provided configurations keys")
	}
	sharedConfigType := reflect.TypeOf(a.Common.SharedRunConfig)
	fields := reflect.VisibleFields(sharedConfigType)
	// Iterate for each field in common(shared configurations).
	for _, field := range fields {
		val := reflect.ValueOf(a.Common.SharedRunConfig).FieldByName(field.Name)
		// Iterate for each app's configurations.
		for i := range a.Apps {
			appVal := reflect.ValueOf(a.Apps[i].RunConfig.SharedRunConfig).FieldByName(field.Name)
			_, ok := appsKeys[i][field.Name]
			// apppVal is the default value for the type and "ok" is for checking if this default value is not set explicitly in the app's configuration.
			if appVal.IsZero() && !ok {
				// Here FieldByName always returns a valid value, it can also be zero but the field always exists.
				reflect.ValueOf(&a.Apps[i].RunConfig.SharedRunConfig).
					Elem().
					FieldByName(field.Name).
					Set(val)
			}
		}
	}
	for i := range a.Apps {
		if a.Apps[i].AppID == "" {
			a.Apps[i].AppID = filepath.Dir(a.Apps[i].AppDirPath)
		}
	}
	return a.Apps, nil
}

// getAppsKeysFromYaml returns a list of maps with key as the field name and value as the type of the field.
// It is used in getting the configured keys from the YAML file for the apps.
// It is needed as it might be possible that the user wants to provide a default value for one of the app's key.
func (a *RunFileConfig) getAppsKeysFromYaml(bytes []byte) ([]map[string]string, error) {
	result := make([]map[string]string, 0)
	tempMap := make(map[string]interface{})
	err := yaml.Unmarshal(bytes, &tempMap)
	if err != nil {
		return nil, fmt.Errorf("error in parsing the provided app config file to extract provided configurations: %w", err)
	}
	apps := tempMap[APPS].([]interface{})
	for _, app := range apps {
		keyMaps := make(map[string]string)
		for k, v := range app.(map[interface{}]interface{}) {
			keyMaps[k.(string)] = reflect.TypeOf(v).String()
		}
		result = append(result, keyMaps)
	}
	return result, nil
}

// resolvePathToAbsAndValidate resolves the relative paths in run file to absolute path and validates the file path.
func (a *RunFileConfig) resolvePathToAbsAndValidate(baseDir string, paths ...*string) error {
	var err error
	for _, path := range paths {
		if *path == "" {
			continue
		}
		absPath := utils.GetAbsPath(baseDir, *path)
		if err != nil {
			return err
		}
		*path = absPath
		if err = utils.ValidateFilePaths(*path); err != nil {
			return err
		}
	}
	return nil
}
