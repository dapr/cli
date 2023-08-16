/*
Copyright 2021 The Dapr Authors
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
	"strconv"
	"strings"
	"time"

	ps "github.com/mitchellh/go-ps"
	process "github.com/shirou/gopsutil/process"

	"github.com/dapr/dapr/pkg/runtime"

	"github.com/dapr/cli/pkg/age"
	"github.com/dapr/cli/pkg/metadata"
	"github.com/dapr/cli/utils"
)

// ListOutput represents the application ID, application port and creation time.
type ListOutput struct {
	AppID              string `csv:"APP ID"    json:"appId"              yaml:"appId"`
	HTTPPort           int    `csv:"HTTP PORT" json:"httpPort"           yaml:"httpPort"`
	GRPCPort           int    `csv:"GRPC PORT" json:"grpcPort"           yaml:"grpcPort"`
	AppPort            int    `csv:"APP PORT"  json:"appPort"            yaml:"appPort"`
	MetricsEnabled     bool   `csv:"-"         json:"metricsEnabled"     yaml:"metricsEnabled"` // Not displayed in table, consumed by dashboard.
	Command            string `csv:"COMMAND"   json:"command"            yaml:"command"`
	Age                string `csv:"AGE"       json:"age"                yaml:"age"`
	Created            string `csv:"CREATED"   json:"created"            yaml:"created"`
	DaprdPID           int    `csv:"DAPRD PID" json:"daprdPid"           yaml:"daprdPid"`
	CliPID             int    `csv:"CLI PID"   json:"cliPid"             yaml:"cliPid"`
	AppPID             int    `csv:"APP PID"   json:"appPid"             yaml:"appPid"`
	MaxRequestBodySize int    `csv:"-"         json:"maxRequestBodySize" yaml:"maxRequestBodySize"` // Additional field, not displayed in table.
	HTTPReadBufferSize int    `csv:"-"         json:"httpReadBufferSize" yaml:"httpReadBufferSize"` // Additional field, not displayed in table.
	RunTemplatePath    string `csv:"RUN_TEMPLATE_PATH"  json:"runTemplatePath"            yaml:"runTemplatePath"`
	AppLogPath         string `csv:"APP_LOG_PATH"  json:"appLogPath"            yaml:"appLogPath"`
	DaprDLogPath       string `csv:"DAPRD_LOG_PATH"  json:"daprdLogPath"            yaml:"daprdLogPath"`
	RunTemplateName    string `json:"runTemplateName"            yaml:"runTemplateName"` // specifically omitted in csv output.
}

func (d *daprProcess) List() ([]ListOutput, error) {
	return List()
}

// List outputs all the applications.
func List() ([]ListOutput, error) {
	list := []ListOutput{}

	processes, err := ps.Processes()
	if err != nil {
		return nil, err
	}

	// Populates the map if all data is available for the sidecar.
	for _, proc := range processes {
		executable := strings.ToLower(proc.Executable())
		if (executable == "daprd") || (executable == "daprd.exe") {
			procDetails, err := process.NewProcess(int32(proc.Pid()))
			if err != nil {
				continue
			}

			cmdLine, err := procDetails.Cmdline()
			if err != nil {
				continue
			}

			cmdLineItems := strings.Fields(cmdLine)
			if len(cmdLineItems) <= 1 {
				continue
			}

			// Parse command line arguments, example format for cmdLine `daprd --flag1 value1 --enable-flag2 --flag3 value3`.
			argumentsMap := make(map[string]string)
			for i := 1; i < len(cmdLineItems)-1; {
				if !strings.HasPrefix(cmdLineItems[i+1], "--") {
					argumentsMap[cmdLineItems[i]] = cmdLineItems[i+1]
					i += 2
				} else {
					argumentsMap[cmdLineItems[i]] = ""
					i++
				}
			}

			httpPort := getIntArg(argumentsMap, "--dapr-http-port", runtime.DefaultDaprHTTPPort)

			grpcPort := getIntArg(argumentsMap, "--dapr-grpc-port", runtime.DefaultDaprAPIGRPCPort)

			appPort := getIntArg(argumentsMap, "--app-port", 0)

			enableMetrics, err := strconv.ParseBool(argumentsMap["--enable-metrics"])
			if err != nil {
				// Default is true for metrics.
				enableMetrics = true
			}

			maxRequestBodySize := getIntArg(argumentsMap, "--dapr-http-max-request-size", runtime.DefaultMaxRequestBodySize)

			httpReadBufferSize := getIntArg(argumentsMap, "--dapr-http-read-buffer-size", runtime.DefaultReadBufferSize)

			appID := argumentsMap["--app-id"]
			appCmd := ""
			appPIDString := ""
			cliPIDString := ""
			runTemplatePath := ""
			appLogPath := ""
			daprdLogPath := ""
			runTemplateName := ""
			socket := argumentsMap["--unix-domain-socket"]
			appMetadata, err := metadata.Get(httpPort, appID, socket)
			if err == nil {
				appCmd = appMetadata.Extended["appCommand"]
				appPIDString = appMetadata.Extended["appPID"]
				cliPIDString = appMetadata.Extended["cliPID"]
				runTemplatePath = appMetadata.Extended["runTemplatePath"]
				runTemplateName = appMetadata.Extended["runTemplateName"]
				appLogPath = appMetadata.Extended["appLogPath"]
				daprdLogPath = appMetadata.Extended["daprdLogPath"]
			}

			appPID, err := strconv.Atoi(appPIDString)
			if err != nil {
				appPID = 0
			}

			// Parse functions return an error on bad input.
			cliPID, err := strconv.Atoi(cliPIDString)
			if err != nil {
				cliPID = 0
			}

			daprPID := proc.Pid()

			createUnixTimeMilliseconds, err := procDetails.CreateTime()
			if err != nil {
				continue
			}

			createTime := time.Unix(createUnixTimeMilliseconds/1000, 0)

			listRow := ListOutput{
				Created:            createTime.Format("2006-01-02 15:04.05"),
				Age:                age.GetAge(createTime),
				DaprdPID:           daprPID,
				CliPID:             cliPID,
				AppID:              appID,
				AppPID:             appPID,
				HTTPPort:           httpPort,
				GRPCPort:           grpcPort,
				AppPort:            appPort,
				MetricsEnabled:     enableMetrics,
				Command:            utils.TruncateString(appCmd, 20),
				MaxRequestBodySize: maxRequestBodySize,
				HTTPReadBufferSize: httpReadBufferSize,
				RunTemplatePath:    runTemplatePath,
				RunTemplateName:    runTemplateName,
				AppLogPath:         appLogPath,
				DaprDLogPath:       daprdLogPath,
			}

			// filter only dashboard instance.
			if listRow.AppID != "" {
				list = append(list, listRow)
			}
		}
	}

	return list, nil
}

// getIntArg returns the value of the argument as an integer.
// If the argument is not set, or is not an integer, it returns the default value.
func getIntArg(argMap map[string]string, argKey string, argDef int) int {
	if arg, ok := argMap[argKey]; ok {
		if argInt, err := strconv.Atoi(arg); err == nil {
			return argInt
		}
	}
	return argDef
}

// GetCLIPIDCountMap returns a map of CLI PIDs to number of apps started with it.
func GetCLIPIDCountMap(apps []ListOutput) map[int]int {
	cliPIDCountMap := make(map[int]int, len(apps))
	for _, app := range apps {
		cliPIDCountMap[app.CliPID]++
	}
	return cliPIDCountMap
}
