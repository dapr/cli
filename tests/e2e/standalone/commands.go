//go:build e2e
// +build e2e

/*
Copyright 2022 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package standalone_test

import (
	"github.com/dapr/cli/tests/e2e/common"
	"github.com/dapr/cli/tests/e2e/spawn"
)

// cmdInit installs Dapr with the init command and returns the command output and error.
//
// When DAPR_E2E_INIT_SLIM is true, it will install Dapr without Docker containers.
// This is useful for scenarios where Docker containers are not available, e.g.,
// in GitHub actions Windows runner.
//
// Arguments to the init command can be passed via args.
func cmdInit(runtimeVersion string, args ...string) (string, error) {
	args = append([]string{"init", "--log-as-json", "--runtime-version", runtimeVersion}, args...)

	if isSlimMode() {
		args = append(args, "--slim")
	}

	return spawn.Command(common.GetDaprPath(), args...)
}

// cmdUninstall uninstalls Dapr with --all flag and returns the command output and error.
func cmdUninstall() (string, error) {
	return spawn.Command(common.GetDaprPath(), "uninstall", "--log-as-json", "--all")
}

// cmdPublish publishes a message to the specified pubsub and topic, and returns the command output and error.
func cmdPublish(appId, pubsub, topic, unixDomainSocket string, args ...string) (string, error) {
	args = append([]string{"publish", "--log-as-json", "--publish-app-id", appId, "--pubsub", pubsub, "--topic", topic}, args...)

	if unixDomainSocket != "" {
		args = append(args, "--unix-domain-socket", unixDomainSocket)
	}

	return spawn.Command(common.GetDaprPath(), args...)
}

// cmdStop stops the specified app and returns the command output and error.
func cmdStop(appId string) (string, error) {
	return spawn.Command(common.GetDaprPath(), "stop", "--log-as-json", "--app-id", appId)
}
