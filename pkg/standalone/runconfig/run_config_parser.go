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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/dapr/cli/utils"

	"gopkg.in/yaml.v2"
)

// constants for the keys from the yaml file.
const APPS = "apps"

func (a *AppsRunConfig) ParseAppsConfig(configFile string) ([]map[string]string, error) {
	bytes, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error in reading the provided app config file: %w", err)
	}
	err = yaml.Unmarshal(bytes, &a)
	if err != nil {
		return nil, fmt.Errorf("error in parsing the provided app config file: %w", err)
	}
	keyMappings, err := a.getKeyMappingFromYaml(bytes)
	if err != nil {
		return nil, err
	}
	return keyMappings, nil
}

func (a *AppsRunConfig) ValidateRunConfig() error {
	if a.Version == 0 {
		return errors.New("required field 'version' not found in the provided run template file")
	}
	// validate all paths in commons.
	err := utils.ValidateFilePaths(a.Common.ConfigFile, a.Common.ResourcesPath)
	if err != nil {
		return err
	}
	for i := 0; i < len(a.Apps); i++ {
		if a.Apps[i].AppDirPath == "" {
			return errors.New("required filed 'app_dir_path' not found in the provided app config file")
		}
		// validate all paths in apps.
		err := utils.ValidateFilePaths(a.Apps[i].ConfigFile, a.Apps[i].ResourcesPath, a.Apps[i].AppDirPath)
		if err != nil {
			return err
		}
	}
	return nil
}

// getKeyMappingFromYaml returns a list of maps with key as the field name and value as the type of the field.
// it is used in getting the configured keys from the yaml file for the apps.
func (a *AppsRunConfig) getKeyMappingFromYaml(bytes []byte) ([]map[string]string, error) {
	result := make([]map[string]string, 0)
	tempMap := make(map[string]interface{})
	err := yaml.Unmarshal(bytes, &tempMap)
	if err != nil {
		return nil, fmt.Errorf("error in parsing the provided app config file to extract provided parameters: %w", err)
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

// GetApps returns a list of apps with the merged values fopr the keys from common section of the yaml file.
func (a *AppsRunConfig) GetApps(keyMappings []map[string]string) []Apps {
	// get a mapping of parsed values from the yaml file for the common section.
	sharedRunConfigMapChan := make(chan map[string]reflect.Value)
	go a.getSharedRunConfigMap(sharedRunConfigMapChan)

	// get a list of maps with key as the field name and value as the reflect value of the field for the apps section of the yaml file.
	appRunConfigListChan := make(chan []map[string]reflect.Value)
	go a.getAppRunConfigList(appRunConfigListChan)

	var sharedRunConfigMap map[string]reflect.Value
	var appRunConfigList []map[string]reflect.Value

	for i := 0; i < 2; i++ {
		select {
		case sharedRunConfigMap = <-sharedRunConfigMapChan:
		case appRunConfigList = <-appRunConfigListChan:
		}
	}

	// merge appRunConfigList and sharedRunConfigMap only if that field is not set in the apps section of the yaml file.
	for index, appRunConfigMap := range appRunConfigList {
		for key, value := range appRunConfigMap {
			if _, exist := keyMappings[index][key]; !exist {
				if value.IsZero() {
					if val, ok := sharedRunConfigMap[key]; ok {
						appRunConfigMap[key] = val
					}
				}
			}
		}
	}

	// set the merged values in the Apps struct.
	for i, appRunConfigMap := range appRunConfigList {
		for key, value := range appRunConfigMap {
			switch key {
			case "ConfigFile":
				a.Apps[i].RunConfig.SharedRunConfig.ConfigFile = value.Interface().(string)
			case "AppProtocol":
				a.Apps[i].RunConfig.SharedRunConfig.AppProtocol = value.Interface().(string)
			case "APIListenAddresses":
				a.Apps[i].RunConfig.SharedRunConfig.APIListenAddresses = value.Interface().(string)
			case "EnableProfiling":
				a.Apps[i].RunConfig.SharedRunConfig.EnableProfiling = value.Interface().(bool)
			case "LogLevel":
				a.Apps[i].RunConfig.SharedRunConfig.LogLevel = value.Interface().(string)
			case "MaxConcurrency":
				a.Apps[i].RunConfig.SharedRunConfig.MaxConcurrency = value.Interface().(int)
			case "PlacementHostAddr":
				a.Apps[i].RunConfig.SharedRunConfig.PlacementHostAddr = value.Interface().(string)
			case "ResourcesPath":
				a.Apps[i].RunConfig.SharedRunConfig.ResourcesPath = value.Interface().(string)
			case "ComponentsPath":
				a.Apps[i].RunConfig.SharedRunConfig.ComponentsPath = value.Interface().(string)
			case "AppSSL":
				a.Apps[i].RunConfig.SharedRunConfig.AppSSL = value.Interface().(bool)
			case "MaxRequestBodySize":
				a.Apps[i].RunConfig.SharedRunConfig.MaxRequestBodySize = value.Interface().(int)
			case "HTTPReadBufferSize":
				a.Apps[i].RunConfig.SharedRunConfig.HTTPReadBufferSize = value.Interface().(int)
			case "EnableAppHealth":
				a.Apps[i].RunConfig.SharedRunConfig.EnableAppHealth = value.Interface().(bool)
			case "AppHealthPath":
				a.Apps[i].RunConfig.SharedRunConfig.AppHealthPath = value.Interface().(string)
			case "AppHealthInterval":
				a.Apps[i].RunConfig.SharedRunConfig.AppHealthInterval = value.Interface().(int)
			case "AppHealthTimeout":
				a.Apps[i].RunConfig.SharedRunConfig.AppHealthTimeout = value.Interface().(int)
			case "AppHealthThreshold":
				a.Apps[i].RunConfig.SharedRunConfig.AppHealthThreshold = value.Interface().(int)
			case "EnableAPILogging":
				a.Apps[i].RunConfig.SharedRunConfig.EnableAPILogging = value.Interface().(bool)
			}
		}
	}
	return a.Apps
}

// getSharedRunConfigMap returns a map with key as the field name and value as the reflect value of the field for the common section of the yaml file.
func (a *AppsRunConfig) getSharedRunConfigMap(sharedRunConfigMapChan chan map[string]reflect.Value) {
	sharedRunConfigMap := make(map[string]reflect.Value)
	sharedConfigSchema := reflect.ValueOf(a.Common.SharedRunConfig)
	for i := 0; i < sharedConfigSchema.NumField(); i++ {
		valueField := sharedConfigSchema.Field(i).Interface()
		typeField := sharedConfigSchema.Type().Field(i)
		sharedRunConfigMap[typeField.Name] = reflect.ValueOf(valueField)
	}
	sharedRunConfigMapChan <- sharedRunConfigMap
}

// getAppRunConfigList returns a list of maps with key as the field name and value as the reflect value of the field for the apps section of the yaml file.
func (a *AppsRunConfig) getAppRunConfigList(appRunConfigListChan chan []map[string]reflect.Value) {
	appRunConfigList := make([]map[string]reflect.Value, 0)
	for j := 0; j < len(a.Apps); j++ {
		// set appID to appDir if not provided.
		if a.Apps[j].AppID == "" {
			a.Apps[j].AppID = filepath.Dir(a.Apps[j].AppDirPath)
		}
		appSchema := reflect.ValueOf(a.Apps[j].RunConfig.SharedRunConfig)
		appRunConfigMap := make(map[string]reflect.Value)
		for i := 0; i < appSchema.NumField(); i++ {
			valueField := appSchema.Field(i).Interface()
			typeField := appSchema.Type().Field(i)
			appRunConfigMap[typeField.Name] = reflect.ValueOf(valueField)
		}
		appRunConfigList = append(appRunConfigList, appRunConfigMap)
	}
	appRunConfigListChan <- appRunConfigList
}
