// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/dapr/cli/pkg/print"

	"github.com/briandowns/spinner"
)

const daprManifestPath = "https://daprreleases.blob.core.windows.net/manifest/dapr-operator.yaml"

// Initialize to deploy the Dapr operator
func Init() error {
	kubeExists := kubeconfigExists()
	if !kubeExists {
		return errors.New("Can't connect to a Kubernetes cluster. Make sure you have the Kubernetes config file on your machine")
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

	err := runCmdAndWait("kubectl", "apply", "-f", daprManifestPath)
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

func kubeconfigExists() bool {
	_, err := Client()
	return err == nil
}

func runCmdAndWait(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	err := cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
