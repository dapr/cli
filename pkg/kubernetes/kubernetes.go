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
	cli_ver "github.com/dapr/cli/pkg/version"
	"github.com/dapr/cli/utils"

	"github.com/briandowns/spinner"
	scheme "github.com/dapr/dapr/pkg/client/clientset/versioned"
)

// Init deploys the Dapr operator
func Init(namespace string) error {
	client, err := DaprClient()
	if err != nil {
		return fmt.Errorf("can't connect to a Kubernetes cluster: %v", err)
	}

	version, err := cli_ver.GetLatestRelease(cli_ver.DaprGitHubOrg, cli_ver.DaprGitHubRepo)
	if err != nil {
		return fmt.Errorf("cannot get the manifest file: %s", err)
	}

	var daprManifestPath string = "https://github.com/dapr/dapr/releases/download/" + version + "/dapr-operator.yaml"

	msg := fmt.Sprintf("Deploying the Dapr Operator to your cluster in namespace %s", namespace)
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

	_, err = utils.RunCmdAndWait("kubectl", "apply", "-f", daprManifestPath, "--namespace="+namespace)
	if err != nil {
		if s != nil {
			s.Stop()
		}
		return err
	}

	err = installConfig(client, namespace)
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

// installConfig installs a configuration resource of a custom CRD called configurations.dapr.io
func installConfig(client scheme.Interface, namespace string) error {
	config := GetDefaultConfiguration()
	_, err := client.ConfigurationV1alpha1().Configurations(namespace).Create(&config)
	return err
}
