// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"github.com/dapr/cli/pkg/age"
	"github.com/dapr/cli/pkg/rundata"
	"github.com/dapr/cli/utils"
	ps "github.com/mitchellh/go-ps"
)

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

func List() ([]ListOutput, error) {
	list := []ListOutput{}

	runtimeData, err := rundata.ReadAllRunData()
	if err != nil {
		return nil, err
	}

	for _, runtimeLine := range *runtimeData {
		proc, err := ps.FindProcess(runtimeLine.PID)
		if proc == nil && err == nil {
			continue
		}

		// TODO: Call to /metadata and validate the runtime data
		var listRow = ListOutput{
			AppID:    runtimeLine.AppId,
			HTTPPort: runtimeLine.DaprHTTPPort,
			GRPCPort: runtimeLine.DaprGRPCPort,
			Command:  utils.TruncateString(runtimeLine.Command, 20),
			Created:  runtimeLine.Created.Format("2006-01-02 15:04.05"),
			PID:      runtimeLine.PID,
		}
		if runtimeLine.AppPort > 0 {
			listRow.AppPort = runtimeLine.AppPort
		}
		listRow.Age = age.GetAge(runtimeLine.Created)
		list = append(list, listRow)
	}

	return list, nil
}
