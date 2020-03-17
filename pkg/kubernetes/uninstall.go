// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"errors"
	"fmt"

	pkg_version "github.com/dapr/cli/pkg/version"
	"github.com/dapr/cli/utils"
)

// Uninstall removes Dapr
func Uninstall() error {
	version, err := pkg_version.GetLatestRelease(pkg_version.DaprGitHubOrg, pkg_version.DaprGitHubRepo)
	if err != nil {
		return fmt.Errorf("cannot get the manifest file: %s", err)
	}

	var daprManifestPath string = "https://github.com/dapr/dapr/releases/download/" + version + "/dapr-operator.yaml"

	_, err = utils.RunCmdAndWait("kubectl", "delete", "-f", daprManifestPath)
	if err != nil {
		return errors.New("is Dapr running? uninstall does not remove Dapr when installed via Helm")
	}

	return nil
}
