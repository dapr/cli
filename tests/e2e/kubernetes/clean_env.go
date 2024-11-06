//go:build e2e || templatek8s
// +build e2e templatek8s

/*
Copyright 2021 The Dapr Authors
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

package kubernetes_test

import (
	"strings"
	"testing"

	"github.com/dapr/cli/tests/e2e/common"
)

var (
	clusterRoles1_9_X         = []string{"dapr-operator-admin", "dashboard-reader"}
	clusterRoleBindings1_9_X  = []string{"dapr-operator", "dapr-role-tokenreview-binding", "dashboard-reader-global"}
	clusterRoles1_10_X        = []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"}
	clusterRoleBindings1_10_X = []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"}
)

// ensureCleanEnv function needs to be called in every Test function.
// sets necessary variable values and uninstalls any previously installed `dapr`.
func ensureCleanEnv(t *testing.T, useDaprLatestVersion bool) {
	common.EnsureEnvVersionSet(t, useDaprLatestVersion)

	if strings.HasPrefix(common.CurrentVersionDetails.RuntimeVersion, "1.9.") {
		common.CurrentVersionDetails.ClusterRoles = clusterRoles1_9_X
		common.CurrentVersionDetails.ClusterRoleBindings = clusterRoleBindings1_9_X
	} else {
		common.CurrentVersionDetails.ClusterRoles = clusterRoles1_10_X
		common.CurrentVersionDetails.ClusterRoleBindings = clusterRoleBindings1_10_X
	}
	// Ensure a clean environment
	common.EnsureUninstall(true, true) // does not wait for pod deletion
}
