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
	err = a.resolvePathToAbsAndValidate(baseDir, &a.Common.ConfigFile, &a.Common.ResourcesPath, &a.Common.DaprdInstallPath)
	if err != nil {
		return err
	}

	// Resolves common's section ResourcesPaths to absolute paths and validates them.
	for i := range a.Common.ResourcesPaths {
		err := a.resolvePathToAbsAndValidate(baseDir, &a.Common.ResourcesPaths[i])
		if err != nil {
			return err
		}
	}

	// Merge common's section ResourcesPaths and ResourcePath. ResourcesPaths will be single source of truth for resources to be loaded.
	if len(strings.TrimSpace(a.Common.ResourcesPath)) > 0 {
		a.Common.ResourcesPaths = append(a.Common.ResourcesPaths, a.Common.ResourcesPath)
	}

	for i := 0; i < len(a.Apps); i++ {
		if a.Apps[i].AppDirPath == "" {
			return errors.New("required field 'appDirPath' not found in the provided app config file")
		}
		// It resolves the relative AppDirPath to absolute path and validates it.
		err := a.resolvePathToAbsAndValidate(baseDir, &a.Apps[i].AppDirPath)
		if err != nil {
			return err
		}
		// All other relative paths present inside the specific app's in the YAML file, should be resolved relative to AppDirPath for that app.
		err = a.resolvePathToAbsAndValidate(a.Apps[i].AppDirPath, &a.Apps[i].ConfigFile, &a.Apps[i].ResourcesPath, &a.Apps[i].DaprdInstallPath)
		if err != nil {
			return err
		}

		// Resolves ResourcesPaths to absolute paths and validates them.
		for j := range a.Apps[i].ResourcesPaths {
			err := a.resolvePathToAbsAndValidate(a.Apps[i].AppDirPath, &a.Apps[i].ResourcesPaths[j])
			if err != nil {
				return err
			}
		}

		// Merge app's section ResourcesPaths and ResourcePath. ResourcesPaths will be single source of truth for resources to be loaded.
		if len(strings.TrimSpace(a.Apps[i].ResourcesPath)) > 0 {
			a.Apps[i].ResourcesPaths = append(a.Apps[i].ResourcesPaths, a.Apps[i].ResourcesPath)
		}
	}
	return nil
}

// GetApps orchestrates the parsing of supplied run file, validating fields and consolidating SharedRunConfig for the apps.
// It returns a list of apps with the merged values for the SharedRunConfig from common section of the YAML file.
func (a *RunFileConfig) GetApps(runFilePath string) ([]App, error) {
	err := a.parseAppsConfig(runFilePath)
	if err != nil {
		return nil, err
	}
	err = a.validateRunConfig(runFilePath)
	if err != nil {
		return nil, err
	}
	err = a.resolveResourcesAndConfigFilePaths()
	if err != nil {
		return nil, err
	}
	a.mergeCommonAndAppsSharedRunConfig()
	a.mergeCommonAndAppsEnv()

	// Set and validates default fields in the run file.
	err = a.setDefaultFields()
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

// setDefaultFields sets the default values for the fields that are not provided in the run file.
func (a *RunFileConfig) setDefaultFields() error {
	for i := range a.Apps {
		if err := a.setAppIDIfEmpty(&a.Apps[i]); err != nil {
			return err
		}
		if err := a.setAndValidateLogDestination(&a.Apps[i]); err != nil {
			return err
		}
	}
	return nil
}

// Set AppID to the directory name of appDirPath.
// appDirPath is a mandatory field in the run file and at this point it is already validated and resolved to its absolute path.
func (a *RunFileConfig) setAppIDIfEmpty(app *App) error {
	if app.AppID == "" {
		basePath, err := a.getBasePathFromAbsPath(app.AppDirPath)
		if err != nil {
			return fmt.Errorf("error in setting the app id: %w", err)
		}
		app.AppID = basePath
	}
	return nil
}

// setAndValidateLogDestination sets the default log destination if not provided in the run file.
// It also validates the log destination if provided.
func (a *RunFileConfig) setAndValidateLogDestination(app *App) error {
	if app.DaprdLogDestination == "" {
		app.DaprdLogDestination = DefaultDaprdLogDest
	} else if err := app.DaprdLogDestination.IsValid(); err != nil {
		return err
	}
	if app.AppLogDestination == "" {
		app.AppLogDestination = DefaultAppLogDest
	} else if err := app.AppLogDestination.IsValid(); err != nil {
		return err
	}
	return nil
}

// Gets the base path from the absolute path of the appDirPath.
func (a *RunFileConfig) getBasePathFromAbsPath(appDirPath string) (string, error) {
	if filepath.IsAbs(appDirPath) {
		return filepath.Base(appDirPath), nil
	}
	return "", fmt.Errorf("error in getting the base path from the provided appDirPath %q: ", appDirPath)
}

// resolvePathToAbsAndValidate resolves the relative paths in run file to absolute path and validates the file path.
func (a *RunFileConfig) resolvePathToAbsAndValidate(baseDir string, paths ...*string) error {
	var err error
	for _, path := range paths {
		if *path == "" {
			continue
		}
		*path, err = utils.ResolveHomeDir(*path)
		if err != nil {
			return err
		}
		absPath := utils.GetAbsPath(baseDir, *path)
		if err != nil {
			return err
		}
		*path = absPath
		if err = utils.ValidateFilePath(*path); err != nil {
			return err
		}
	}
	return nil
}

// Resolve resources and config file paths for each app.
func (a *RunFileConfig) resolveResourcesAndConfigFilePaths() error {
	for i := range a.Apps {
		app := &a.Apps[i]
		// Make sure apps's "DaprPathCmdFlag" is updated here as it is used in deciding precedence for resources and config path.
		if app.DaprdInstallPath == "" {
			app.DaprdInstallPath = a.Common.DaprdInstallPath
		}

		err := a.resolveResourcesFilePath(app)
		if err != nil {
			return fmt.Errorf("error in resolving resources path for app %q: %w", app.AppID, err)
		}

		err = a.resolveConfigFilePath(app)
		if err != nil {
			return fmt.Errorf("error in resolving config file path for app %q: %w", app.AppID, err)
		}
	}
	return nil
}

// mergeCommonAndAppsEnv merges env maps from common and individual apps.
// Precedence order for envs -> apps[i].envs > common.envs.
func (a *RunFileConfig) mergeCommonAndAppsEnv() {
	for i := range a.Apps {
		for k, v := range a.Common.Env {
			if _, ok := a.Apps[i].Env[k]; !ok {
				a.Apps[i].Env[k] = v
			}
		}
	}
}

// resolveResourcesFilePath resolves the resources path for the app.
// Precedence order for resourcesPaths -> apps[i].resourcesPaths > apps[i].appDirPath/.dapr/resources > common.resourcesPaths > dapr default resources path.
func (a *RunFileConfig) resolveResourcesFilePath(app *App) error {
	if len(app.ResourcesPaths) > 0 {
		return nil
	}
	localResourcesDir := filepath.Join(app.AppDirPath, standalone.DefaultDaprDirName, standalone.DefaultResourcesDirName)
	if err := utils.ValidateFilePath(localResourcesDir); err == nil {
		app.ResourcesPaths = []string{localResourcesDir}
	} else if len(a.Common.ResourcesPaths) > 0 {
		app.ResourcesPaths = append(app.ResourcesPaths, a.Common.ResourcesPaths...)
	} else {
		daprDirPath, err := standalone.GetDaprRuntimePath(app.DaprdInstallPath)
		if err != nil {
			return fmt.Errorf("error getting dapr install path: %w", err)
		}
		app.ResourcesPaths = []string{standalone.GetDaprComponentsPath(daprDirPath)}
	}
	return nil
}

// resolveConfigFilePath resolves the config file path for the app.
// Precedence order for configFile -> apps[i].configFile > apps[i].appDirPath/.dapr/config.yaml > common.configFile > dapr default config file.
func (a *RunFileConfig) resolveConfigFilePath(app *App) error {
	if app.ConfigFile != "" {
		return nil
	}
	localConfigFile := filepath.Join(app.AppDirPath, standalone.DefaultDaprDirName, standalone.DefaultConfigFileName)
	if err := utils.ValidateFilePath(localConfigFile); err == nil {
		app.ConfigFile = localConfigFile
	} else if len(strings.TrimSpace(a.Common.ConfigFile)) > 0 {
		app.ConfigFile = a.Common.ConfigFile
	} else {
		daprDirPath, err := standalone.GetDaprRuntimePath(app.DaprdInstallPath)
		if err != nil {
			return fmt.Errorf("error getting dapr install path: %w", err)
		}
		app.ConfigFile = standalone.GetDaprConfigPath(daprDirPath)
	}
	return nil
}
