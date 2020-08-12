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
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	daprLatestVersion = "vlatest"
)

// Init deploys the Dapr operator using the supplied runtime version.
func Init(version string) error {
	client, err := DaprClient()
	if err != nil {
		return fmt.Errorf("can't connect to a Kubernetes cluster: %v", err)
	}

	if version == daprLatestVersion {
		var v string
		v, err = cli_ver.GetLatestRelease(cli_ver.DaprGitHubOrg, cli_ver.DaprGitHubRepo)
		if err != nil {
			return fmt.Errorf("cannot get the manifest file: %s", err)
		}

		version = v
	}

	var daprManifestPath string = "https://github.com/dapr/dapr/releases/download/" + version + "/dapr-operator.yaml"

	msg := "Deploying the Dapr control plane to your cluster..."
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

	err = installConfig(client)
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
func installConfig(client scheme.Interface) error {
	config := GetDefaultConfiguration()
	_, err := client.ConfigurationV1alpha1().Configurations(meta_v1.NamespaceDefault).Create(&config)
	return err
}
