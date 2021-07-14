// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"strconv"
	"strings"
	"time"

	"github.com/dapr/cli/pkg/age"
	"github.com/dapr/cli/pkg/metadata"
	"github.com/dapr/cli/utils"
	"github.com/dapr/dapr/pkg/runtime"
	ps "github.com/mitchellh/go-ps"
	process "github.com/shirou/gopsutil/process"
)

// ListOutput represents the application ID, application port and creation time.
type ListOutput struct {
	AppID          string `csv:"APP ID"    json:"appId"          yaml:"appId"`
	HTTPPort       int    `csv:"HTTP PORT" json:"httpPort"       yaml:"httpPort"`
	GRPCPort       int    `csv:"GRPC PORT" json:"grpcPort"       yaml:"grpcPort"`
	AppPort        int    `csv:"APP PORT"  json:"appPort"        yaml:"appPort"`
	MetricsEnabled bool   `csv:"-"         json:"metricsEnabled" yaml:"metricsEnabled"` // Not displayed in table, consumed by dashboard.
	Command        string `csv:"COMMAND"   json:"command"        yaml:"command"`
	Age            string `csv:"AGE"       json:"age"            yaml:"age"`
	Created        string `csv:"CREATED"   json:"created"        yaml:"created"`
	PID            int    `csv:"PID"       json:"pid"            yaml:"pid"`
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

	// Populates the list if all data is available for the sidecar.
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

			argumentsMap := make(map[string]string)
			for i := 1; i < len(cmdLineItems)-1; i += 2 {
				argumentsMap[cmdLineItems[i]] = cmdLineItems[i+1]
			}

			httpPort := runtime.DefaultDaprHTTPPort
			if daprHTTPPort, ok := argumentsMap["--dapr-http-port"]; ok {
				if iHttpPort, err := strconv.Atoi(daprHTTPPort); err == nil {
					httpPort = iHttpPort
				}
			}

			grpcPort := runtime.DefaultDaprAPIGRPCPort
			if daprGRPCPort, ok := argumentsMap["--dapr-grpc-port"]; ok {
				if iGrpcPort, err := strconv.Atoi(daprGRPCPort); err == nil {
					grpcPort = iGrpcPort
				}
			}

			appPort, err := strconv.Atoi(argumentsMap["--app-port"])
			if err != nil {
				appPort = 0
			}

			enableMetrics, err := strconv.ParseBool(argumentsMap["--enable-metrics"])
			if err != nil {
				// Default is true for metrics.
				enableMetrics = true
			}
			appID := argumentsMap["--app-id"]
			appCmd := ""
			cliPIDString := ""
			appMetadata, err := metadata.Get(httpPort)
			if err == nil {
				appCmd = appMetadata.Extended["appCommand"]
				cliPIDString = appMetadata.Extended["cliPID"]
			}

			// Parse functions return an error on bad input.
			cliPID, err := strconv.Atoi(cliPIDString)
			if err != nil {
				cliPID = proc.Pid()
			}

			createUnixTimeMilliseconds, err := procDetails.CreateTime()
			if err != nil {
				continue
			}

			createTime := time.Unix(createUnixTimeMilliseconds/1000, 0)

			listRow := ListOutput{
				Created: createTime.Format("2006-01-02 15:04.05"),
				Age:     age.GetAge(createTime),
				PID:     cliPID,
			}

			listRow.AppID = appID
			listRow.HTTPPort = httpPort
			listRow.GRPCPort = grpcPort
			listRow.AppPort = appPort
			listRow.MetricsEnabled = enableMetrics
			listRow.Command = utils.TruncateString(appCmd, 20)

			list = append(list, listRow)
		}
	}

	return list, nil
}
