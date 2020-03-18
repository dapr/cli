// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"os"
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

// runData is a placeholder for collected information linking cli and sidecar.
type runData struct {
	cliPID     int
	sidecarPID int
	grpcPort   int
	httpPort   int
	appPort    int
	appID      string
	appCmd     string
}

// List outputs all the applications.
func List() ([]ListOutput, error) {
	list := []ListOutput{}

	processes, err := ps.Processes()
	if err != nil {
		return nil, err
	}

	// Links a cli PID to the corresponding sidecar Process.
	cliToSidecarMap := make(map[int]*runData)

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
			cliPIDString := ""
			appMetadata, err := metadata.Get(httpPort)
			if err == nil {
				appCmd = appMetadata.Extended["appCommand"]
				cliPIDString = appMetadata.Extended["cliPID"]
			}

			// Parse functions return an error on bad input.
			cliPID, err := strconv.Atoi(cliPIDString)
			if err != nil {
				continue
			}

			var run = runData{
				cliPID:     cliPID,
				sidecarPID: proc.Pid(),
				grpcPort:   grpcPort,
				httpPort:   httpPort,
				appPort:    appPort,
				appID:      appID,
				appCmd:     appCmd,
			}

			cliToSidecarMap[cliPID] = &run
		}
	}

	myPID := os.Getpid()
	// The master list comes from cli processes, even if sidecar is not up.
	for _, proc := range processes {
		executable := strings.ToLower(proc.Executable())
		if (executable == "dapr") || (executable == "dapr.exe") {
			pID := proc.Pid()
			if pID == myPID {
				// Do not display current `dapr list` process.
				continue
			}

			procDetails, err := process.NewProcess(int32(pID))
			if err != nil {
				continue
			}

			createUnixTimeMilliseconds, err := procDetails.CreateTime()
			if err != nil {
				continue
			}

			createTime := time.Unix(createUnixTimeMilliseconds/1000, 0)

			var listRow = ListOutput{
				Created: createTime.Format("2006-01-02 15:04.05"),
				Age:     age.GetAge(createTime),
				PID:     proc.Pid(),
			}

			// Now we use sidecar into to decorate with more info (empty, if sidecar is down).
			run, ok := cliToSidecarMap[proc.Pid()]
			if ok {
				listRow.AppID = run.appID
				listRow.HTTPPort = run.httpPort
				listRow.GRPCPort = run.grpcPort
				listRow.AppPort = run.appPort
				listRow.Command = utils.TruncateString(run.appCmd, 20)
			}

			list = append(list, listRow)
		}
	}

	return list, nil
}
