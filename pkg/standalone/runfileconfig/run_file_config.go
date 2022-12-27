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

import "github.com/dapr/cli/pkg/standalone"

type RunFileConfig struct {
	Common  Common `yaml:"common"`
	Apps    []Apps `yaml:"apps"`
	Version int    `yaml:"version"`
}

type Apps struct {
	standalone.RunConfig `yaml:",inline"`
	AppDirPath           string     `yaml:"app_dir_path"`
	Env                  []EnvItems `yaml:"env"`
}

type Common struct {
	Env                        []EnvItems `yaml:"env"`
	standalone.SharedRunConfig `yaml:",inline"`
}

type EnvItems struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}
