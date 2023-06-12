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
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dapr/cli/pkg/standalone"
)

const (
	appLogFileNamePrefix   = "app"
	daprdLogFileNamePrefix = "daprd"
	logFileExtension       = ".log"
	logsDir                = "logs"
)

// RunFileConfig represents the complete configuration options for the run file.
// It is meant to be used with - "dapr run --run-file <path-to-run-file>" command.
type RunFileConfig struct {
	Common  Common `yaml:"common"`
	Apps    []App  `yaml:"apps"`
	Version int    `yaml:"version"`
	Name    string `yaml:"name,omitempty"`
}

// App represents the configuration options for the apps in the run file.
type App struct {
	standalone.RunConfig `yaml:",inline"`
	AppDirPath           string `yaml:"appDirPath"`
	AppLogFileName       string
	DaprdLogFileName     string
	AppLogWriteCloser    io.WriteCloser
	DaprdLogWriteCloser  io.WriteCloser
}

// Common represents the configuration options for the common section in the run file.
type Common struct {
	standalone.SharedRunConfig `yaml:",inline"`
}

func (a *App) GetLogsDir() string {
	logsPath := filepath.Join(a.AppDirPath, standalone.DefaultDaprDirName, logsDir)
	os.MkdirAll(logsPath, 0o755)
	return logsPath
}

// CreateAppLogFile creates the log file, sets internal file handle
// and returns error if any.
func (a *App) CreateAppLogFile() error {
	var err error
	var f *os.File
	if a.AppLogDestination == standalone.Console {
		f = os.Stdout
	} else {
		f, err = a.createLogFile(appLogFileNamePrefix)
	}
	if err == nil {
		a.AppLogWriteCloser = f
		a.AppLogFileName = f.Name()
	}
	return err
}

// CreateDaprdLogFile creates the log file, sets internal file handle
// and returns error if any.
func (a *App) CreateDaprdLogFile() error {
	var err error
	var f *os.File
	if a.DaprdLogDestination == standalone.Console {
		f = os.Stdout
	} else {
		f, err = a.createLogFile(daprdLogFileNamePrefix)
	}
	if err == nil {
		a.DaprdLogWriteCloser = f
		a.DaprdLogFileName = f.Name()
	}
	return err
}

// createLogFile creates the log file and returns the file handle and error if any.
// It also adds the app ID as a prefix and the current timestamp to the file name as a suffix.
func (a *App) createLogFile(logType string) (*os.File, error) {
	logsPath := a.GetLogsDir()
	fpath := filepath.Join(logsPath, a.AppID+"_"+logType+"_"+time.Now().Format("20060102150405")+logFileExtension)
	f, err := os.Create(fpath)
	return f, err
}

func (a *App) CloseAppLogFile() error {
	if a.AppLogWriteCloser != nil {
		return a.AppLogWriteCloser.Close()
	}
	return nil
}

func (a *App) CloseDaprdLogFile() error {
	if a.DaprdLogWriteCloser != nil {
		return a.DaprdLogWriteCloser.Close()
	}
	return nil
}
