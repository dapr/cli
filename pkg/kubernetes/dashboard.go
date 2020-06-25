// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/briandowns/spinner"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/utils"
)

func InitDashboard() error {

	var dashboardManifestPath string = "https://raw.githubusercontent.com/dapr/dashboard/master/deploy/dashboard.yaml"

	msg := "Deploying Dapr dashboard to your cluster"
	var s *spinner.Spinner

	if runtime.GOOS == "windows" {
		print.InfoStatusEvent(os.Stdout, msg)
	} else {
		s = spinner.New(spinner.CharSets[0], 100*time.Millisecond)
		s.Writer = os.Stdout
		s.Color("cyan")
		s.Suffix = fmt.Sprintf("  %s", msg)
		s.Start()
	}

	_, err := utils.RunCmdAndWait("kubectl", "apply", "-f", dashboardManifestPath)
	if err != nil {
		if s != nil {
			s.Stop()
		}
		return err
	}

	if s != nil {
		s.Stop()
		print.SuccessStatusEvent(os.Stdout, msg)
	}

	return nil
}
