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
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dapr/cli/tests/e2e/common"
	"github.com/dapr/cli/tests/e2e/spawn"
	"github.com/dapr/cli/utils"
)

// cmdDashboard runs the Dapr dashboard and blocks until it is started.
// If the context is done, the dashboard is stopped.
func cmdDashboard(ctx context.Context, port string) error {
	stdOutChan, stdErrChan, err := spawn.CommandWithContext(ctx, common.GetDaprPath(), "dashboard", "--port", port)
	if err != nil {
		return err
	}
	for output := range stdOutChan {
		if strings.Contains(output, "Dapr Dashboard running on") {
			return nil
		}
	}

	errOutput := ""
	for output := range stdErrChan {
		errOutput += output
	}

	return errors.New(fmt.Sprintf("Dashboard could not be started:%v", errOutput))
}

// cmdInit installs Dapr with the init command and returns the command output and error.
//
// When DAPR_E2E_INIT_SLIM is true, it will install Dapr without Docker containers.
// This is useful for scenarios where Docker containers are not available, e.g.,
// in GitHub actions Windows runner.
//
// Arguments to the init command can be passed via args.
func cmdInit(runtimeVersion string, args ...string) (string, error) {
	initArgs := []string{"init", "--log-as-json", "--runtime-version", runtimeVersion}
	daprContainerRuntime := containerRuntime()

	if isSlimMode() {
		initArgs = append(initArgs, "--slim")
	} else if daprContainerRuntime != "" {
		initArgs = append(initArgs, "--container-runtime", daprContainerRuntime)
	}
	initArgs = append(initArgs, args...)

	return spawn.Command(common.GetDaprPath(), initArgs...)
}

// cmdInvoke invokes a method on the specified app and returns the command output and error.
func cmdInvoke(appId, method, unixDomainSocket string, args ...string) (string, error) {
	invokeArgs := []string{"invoke", "--log-as-json", "--app-id", appId, "--method", method}

	if unixDomainSocket != "" {
		invokeArgs = append(invokeArgs, "--unix-domain-socket", unixDomainSocket)
	}

	invokeArgs = append(invokeArgs, args...)

	return spawn.Command(common.GetDaprPath(), invokeArgs...)
}

// cmdList lists the running dapr instances and returns the command output and error.
// format can be empty, "table", "json", or "yaml"
func cmdList(output string) (string, error) {
	args := []string{"list"}

	if output != "" {
		args = append(args, "-o", output)
	}

	return spawn.Command(common.GetDaprPath(), args...)
}

// cmdPublish publishes a message to the specified pubsub and topic, and returns the command output and error.
func cmdPublish(appId, pubsub, topic, unixDomainSocket string, args ...string) (string, error) {
	publishArgs := []string{"publish", "--log-as-json", "--publish-app-id", appId, "--pubsub", pubsub, "--topic", topic}

	if unixDomainSocket != "" {
		publishArgs = append(publishArgs, "--unix-domain-socket", unixDomainSocket)
	}

	publishArgs = append(publishArgs, args...)

	return spawn.Command(common.GetDaprPath(), publishArgs...)
}

// cmdRun runs a Dapr instance and returns the command output and error.
func cmdRun(unixDomainSocket string, args ...string) (string, error) {
	runArgs := []string{"run"}

	if unixDomainSocket != "" {
		runArgs = append(runArgs, "--unix-domain-socket", unixDomainSocket)
	}

	runArgs = append(runArgs, args...)

	return spawn.Command(common.GetDaprPath(), runArgs...)
}

// cmdStop stops the specified app and returns the command output and error.
func cmdStop(appId string, args ...string) (string, error) {
	stopArgs := append([]string{"stop", "--log-as-json", "--app-id", appId}, args...)
	return spawn.Command(common.GetDaprPath(), stopArgs...)
}

// cmdUninstall uninstalls Dapr with --all flag and returns the command output and error.
func cmdUninstall(args ...string) (string, error) {
	uninstallArgs := []string{"uninstall", "--all"}

	daprContainerRuntime := containerRuntime()

	// Add --container-runtime flag only if daprContainerRuntime is not empty, or overridden via args.
	// This is only valid for non-slim mode.
	if !isSlimMode() && daprContainerRuntime != "" && !utils.Contains(args, "--container-runtime") {
		uninstallArgs = append(uninstallArgs, "--container-runtime", daprContainerRuntime)
	}
	uninstallArgs = append(uninstallArgs, args...)

	return spawn.Command(common.GetDaprPath(), uninstallArgs...)
}

// cmdVersion checks the version of Dapr and returns the command output and error.
// output can be empty or "json"
func cmdVersion(output string, args ...string) (string, error) {
	verArgs := []string{"version"}

	if output != "" {
		verArgs = append(verArgs, "-o", output)
	}

	verArgs = append(verArgs, args...)

	return spawn.Command(common.GetDaprPath(), verArgs...)
}
