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
	"io"
	"os"
	"path/filepath"

	"github.com/dapr/cli/pkg/standalone"
)

// RunFileConfig represents the complete configuration options for the run file.
// It is meant to be used with - "dapr run --run-file <path-to-run-file>" command.
type RunFileConfig struct {
	Common  Common `yaml:"common"`
	Apps    []App  `yaml:"apps"`
	Version int    `yaml:"version"`
}

// App represents the configuration options for the apps in the run file.
type App struct {
	standalone.RunConfig `yaml:",inline"`
	AppDirPath           string     `yaml:"app_dir_path"`
	Env                  []EnvItems `yaml:"env"`
	appLogFile           *os.File
	daprdLogFile         *os.File
}

// Common represents the configuration options for the common section in the run file.
type Common struct {
	Env                        []EnvItems `yaml:"env"`
	standalone.SharedRunConfig `yaml:",inline"`
}

// EnvItems represents the env configuration options that are present in commmon and/or individual app's section.
type EnvItems struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func (a *App) GetLogsDir() string {
	logsPath := filepath.Join(a.AppDirPath, ".dapr", "logs")
	if _, err := os.Stat(logsPath); errors.Is(err, os.ErrNotExist) {
		os.MkdirAll(logsPath, 0o755)
	}
	return logsPath
}

func (a *App) GetAppLogFileWriter() (io.WriteCloser, error) {
	logsPath := a.GetLogsDir()
	f, err := os.Create(filepath.Join(logsPath, "app.log"))
	if err != nil {
		a.appLogFile = f
	}
	return f, err
}

func (a *App) GetDaprdLogFileWriter() (io.WriteCloser, error) {
	logsPath := a.GetLogsDir()
	f, err := os.Create(filepath.Join(logsPath, "daprd.log"))
	if err != nil {
		a.daprdLogFile = f
	}
	return f, err
}

func (a *App) CloseAppLogFile() error {
	if a.appLogFile != nil {
		return a.appLogFile.Close()
	}
	return nil
}

func (a *App) CloseDaprdLogFile() error {
	if a.daprdLogFile != nil {
		return a.daprdLogFile.Close()
	}
	return nil
}

func (a *App) GetAppLogFileName() string {
	if a.appLogFile != nil {
		return a.appLogFile.Name()
	}
	return ""
}

func (a *App) GetDaprdLogFileName() string {
	if a.daprdLogFile != nil {
		return a.daprdLogFile.Name()
	}
	return ""
}