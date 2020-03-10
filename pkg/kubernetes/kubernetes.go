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

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/version"
	"github.com/dapr/cli/utils"

	"github.com/briandowns/spinner"
)

var daprManifestPath string = "https://github.com/dapr/dapr/releases/download/v" + version.GetCLIVersion() + "/dapr-operator.yaml"

// Init deploys the Dapr operator
func Init() error {
	_, err := Client()
	if err != nil {
		return fmt.Errorf("can't connect to a Kubernetes cluster: %v", err)
	}

	msg := "Deploying the Dapr Operator to your cluster..."
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

	_, err = utils.RunCmdAndWait("kubectl", "apply", "-f", daprManifestPath)
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
