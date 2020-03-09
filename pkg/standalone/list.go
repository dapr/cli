// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
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
	ps "github.com/mitchellh/go-ps"
	process "github.com/shirou/gopsutil/process"
)

// ListOutput represents the application ID, application port and creation time.
type ListOutput struct {
	AppID    string `csv:"APP ID"`
	HTTPPort int    `csv:"HTTP PORT"`
	GRPCPort int    `csv:"GRPC PORT"`
	AppPort  int    `csv:"APP PORT"`
	Command  string `csv:"COMMAND"`
	Age      string `csv:"AGE"`
	Created  string `csv:"CREATED"`
	PID      int
}

// List outputs all the applications.
func List() ([]ListOutput, error) {
	list := []ListOutput{}

	processes, err := ps.Processes()
	if err != nil {
		return nil, err
	}

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

			createUnixTimeMilliseconds, err := procDetails.CreateTime()
			if err != nil {
				continue
			}
			createTime := time.Unix(createUnixTimeMilliseconds/1000, 0)

			cmdLineItems := strings.Fields(cmdLine)
			if len(cmdLineItems) <= 1 {
				continue
			}

			argumentsMap := make(map[string]string)
			for i := 1; i < len(cmdLineItems)-1; i += 2 {
				argumentsMap[cmdLineItems[i]] = cmdLineItems[i+1]
			}

			httpPort, err := strconv.Atoi(argumentsMap["--dapr-http-port"])
			if err != nil {
				continue
			}

			grpcPort, err := strconv.Atoi(argumentsMap["--dapr-grpc-port"])
			if err != nil {
				continue
			}

			appPort, err := strconv.Atoi(argumentsMap["--app-port"])
			if err != nil {
				appPort = 0
			}

			appID := argumentsMap["--app-id"]
			appCmd := ""
			appMetadata, err := metadata.Get(httpPort)
			if err == nil {
				appCmd = appMetadata.Extended["appCommand"]
			}

			var listRow = ListOutput{
				AppID:    appID,
				HTTPPort: httpPort,
				GRPCPort: grpcPort,
				Command:  utils.TruncateString(appCmd, 20),
				Created:  createTime.Format("2006-01-22 15:04.05"),
				Age:      age.GetAge(createTime),
				PID:      proc.Pid(),
			}

			if appPort > 0 {
				listRow.AppPort = appPort
			}

			list = append(list, listRow)
		}
	}

	return list, nil
}
