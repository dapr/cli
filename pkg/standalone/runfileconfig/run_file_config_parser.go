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
	"strings"

	"github.com/dapr/cli/pkg/standalone"
	"github.com/dapr/cli/utils"

	"gopkg.in/yaml.v2"
)

// Parse the provided run file into a RunFileConfig struct.
func (a *RunFileConfig) parseAppsConfig(runFilePath string) error {
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

// validateRunConfig validates the run file config for mandatory fields.
// It also resolves relative paths to absolute paths and validates them.
func (a *RunFileConfig) validateRunConfig(runFilePath string) error {
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

// GetApps orchestrates the parsing of supplied run file, validating fields and consolidating SharedRunConfig for the apps.
// It returns a list of apps with the merged values for the SharedRunConfig from common section of the YAML file.
func (a *RunFileConfig) GetApps(runFilePath string) ([]Apps, error) {
	err := a.parseAppsConfig(runFilePath)
	if err != nil {
		return nil, err
	}
	err = a.validateRunConfig(runFilePath)
	if err != nil {
		return nil, err
	}
	a.resolveResourcesAndConfigFilePaths()
	a.mergeCommonAndAppsSharedRunConfig()
	a.mergeCommonAndAppsEnv()
	// Resolve app ids if not provided in the run file.
	err = a.setAppIDIfEmpty()
	if err != nil {
		return nil, err
	}
	return a.Apps, nil
}

// mergeCommonAndAppsSharedRunConfig merges the common section of the run file with the apps section.
func (a *RunFileConfig) mergeCommonAndAppsSharedRunConfig() {
	sharedConfigType := reflect.TypeOf(a.Common.SharedRunConfig)
	fields := reflect.VisibleFields(sharedConfigType)
	// Iterate for each field in common(shared configurations).
	for _, field := range fields {
		val := reflect.ValueOf(a.Common.SharedRunConfig).FieldByName(field.Name)
		// Iterate for each app's configurations.
		for i := range a.Apps {
			appVal := reflect.ValueOf(a.Apps[i].RunConfig.SharedRunConfig).FieldByName(field.Name)
			// If apppVal is the default value for the type.
			if appVal.IsZero() {
				// Here FieldByName always returns a valid value, it can also be zero but the field always exists.
				reflect.ValueOf(&a.Apps[i].RunConfig.SharedRunConfig).
					Elem().
					FieldByName(field.Name).
					Set(val)
			}
		}
	}
}

// Set AppID to the directory name of app_dir_path.
// app_dir_path is a mandatory field in the run file and at this point it is already validated and resolved to its absolute path.
func (a *RunFileConfig) setAppIDIfEmpty() error {
	for i := range a.Apps {
		if a.Apps[i].AppID == "" {
			basePath, err := a.getBasePathFromAbsPath(a.Apps[i].AppDirPath)
			if err != nil {
				return err
			}
			a.Apps[i].AppID = basePath
		}
	}
	return nil
}

// Gets the base path from the absolute path of the app_dir_path.
func (a *RunFileConfig) getBasePathFromAbsPath(appDirPath string) (string, error) {
	if filepath.IsAbs(appDirPath) {
		return filepath.Base(appDirPath), nil
	}
	return "", fmt.Errorf("error in getting the base path from the provided app_dir_path %q: ", appDirPath)
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

// Resolve resources and config file paths for each app.
// precedence order for resources_path -> apps[i].resources_path > apps[i].app_dir_path/.dapr/resources > common.resources_path > dapr default resources path.
// precedence order for config_file -> apps[i].config_file > apps[i].app_dir_path/.dapr/config.yaml > common.config_file > dapr default config file.
func (a *RunFileConfig) resolveResourcesAndConfigFilePaths() {
	for i := range a.Apps {
		app := &a.Apps[i]
		// Resolve resources path if not provided in specific app's config.
		if app.ResourcesPath == "" {
			localResourcesDir := filepath.Join(app.AppDirPath, ".dapr", "resources")
			if err := utils.ValidateFilePaths(localResourcesDir); err == nil {
				app.ResourcesPath = localResourcesDir
			} else if len(strings.TrimSpace(a.Common.ResourcesPath)) > 0 {
				app.ResourcesPath = a.Common.ResourcesPath
			} else {
				app.ResourcesPath = standalone.DefaultComponentsDirPath()
			}
		}
		// Resolve config file path if not provided in specific app's config.
		if app.ConfigFile == "" {
			localConfigFile := filepath.Join(app.AppDirPath, ".dapr", "config.yaml")
			if err := utils.ValidateFilePaths(localConfigFile); err == nil {
				app.ConfigFile = localConfigFile
			} else if len(strings.TrimSpace(a.Common.ConfigFile)) > 0 {
				app.ConfigFile = a.Common.ConfigFile
			} else {
				app.ConfigFile = standalone.DefaultConfigFilePath()
			}
		}
	}
}

// mergeCommonAndAppsEnv merges envs from common and individual app section.
// If env is present in both common and app, then the env from app is used.
// If env is present in common but not in app, then the env from common is used.
// If env is present in app but not in common, then the env from app is used.
func (a *RunFileConfig) mergeCommonAndAppsEnv() {
	for i := range a.Common.Env {
		envName := a.Common.Env[i].Name
		for j := range a.Apps {
			envFound := false
			appEnv := a.Apps[j].Env
			for k := range appEnv {
				if envName == appEnv[k].Name {
					envFound = true
					break
				}
			}
			if !envFound {
				a.Apps[j].Env = append(a.Apps[j].Env, a.Common.Env[i])
			}
		}
	}
}
