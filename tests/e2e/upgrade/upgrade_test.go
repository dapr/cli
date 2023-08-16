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

package upgrade

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dapr/cli/tests/e2e/common"
)

type upgradePath struct {
	previous common.VersionDetails
	next     common.VersionDetails
}

var supportedUpgradePaths = []upgradePath{
	{
		// test upgrade on mariner images.
		previous: common.VersionDetails{
			RuntimeVersion:      "1.8.0",
			DashboardVersion:    "0.10.0",
			ImageVariant:        "mariner",
			ClusterRoles:        []string{"dapr-operator-admin", "dashboard-reader"},
			ClusterRoleBindings: []string{"dapr-operator", "dapr-role-tokenreview-binding", "dashboard-reader-global"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io"},
		},
		next: common.VersionDetails{
			RuntimeVersion:      "1.8.7",
			DashboardVersion:    "0.10.0",
			ImageVariant:        "mariner",
			ClusterRoles:        []string{"dapr-operator-admin", "dashboard-reader"},
			ClusterRoleBindings: []string{"dapr-operator", "dapr-role-tokenreview-binding", "dashboard-reader-global"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io"},
		},
	},
	{
		previous: common.VersionDetails{
			RuntimeVersion:      "1.9.5",
			DashboardVersion:    "0.11.0",
			ClusterRoles:        []string{"dapr-operator-admin", "dashboard-reader"},
			ClusterRoleBindings: []string{"dapr-operator", "dapr-role-tokenreview-binding", "dashboard-reader-global"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io"},
		},
		next: common.VersionDetails{
			RuntimeVersion:      "1.10.7",
			DashboardVersion:    "0.12.0",
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io"},
		},
	},
	{
		previous: common.VersionDetails{
			RuntimeVersion:      "1.10.7",
			DashboardVersion:    "0.12.0",
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io"},
		},
		next: common.VersionDetails{
			RuntimeVersion:      "1.11.0",
			DashboardVersion:    "0.13.0",
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
	},
	// test downgrade.
	{
		previous: common.VersionDetails{
			RuntimeVersion:      "1.11.0",
			DashboardVersion:    "0.13.0",
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
		next: common.VersionDetails{
			RuntimeVersion:      "1.10.7",
			DashboardVersion:    "0.12.0",
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io"},
		},
	},
}

func getTestsOnUpgrade(p upgradePath, installOpts, upgradeOpts common.TestOptions) []common.TestCase {
	tests := []common.TestCase{}

	// install previous version.
	tests = append(tests, common.GetTestsOnInstall(p.previous, installOpts)...)

	details := p.next

	tests = append(tests, []common.TestCase{
		{Name: "upgrade to " + details.RuntimeVersion, Callable: common.UpgradeTest(details, upgradeOpts)},
		{Name: "crds exist " + details.RuntimeVersion, Callable: common.CRDTest(details, upgradeOpts)},
		{Name: "clusterroles exist " + details.RuntimeVersion, Callable: common.ClusterRolesTest(details, upgradeOpts)},
		{Name: "clusterrolebindings exist " + details.RuntimeVersion, Callable: common.ClusterRoleBindingsTest(details, upgradeOpts)},
		{Name: "previously applied components exist " + details.RuntimeVersion, Callable: common.ComponentsTestOnInstallUpgrade(upgradeOpts)},
		{Name: "previously applied http endpoints exist " + details.RuntimeVersion, Callable: common.HTTPEndpointsTestOnInstallUpgrade(upgradeOpts)},
		{Name: "check mtls " + details.RuntimeVersion, Callable: common.MTLSTestOnInstallUpgrade(upgradeOpts)},
		{Name: "status check " + details.RuntimeVersion, Callable: common.StatusTestOnInstallUpgrade(details, upgradeOpts)},
	}...)

	// uninstall.
	tests = append(tests, common.GetTestsOnUninstall(p.next, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			// TODO Related to https://github.com/dapr/cli/issues/656
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// delete CRDs if exist.
	tests = append(tests,
		common.TestCase{Name: "delete CRDs " + p.previous.RuntimeVersion, Callable: common.DeleteCRD(p.previous.CustomResourceDefs)},
		common.TestCase{Name: "delete CRDs " + p.next.RuntimeVersion, Callable: common.DeleteCRD(p.next.CustomResourceDefs)})

	return tests
}

// Upgrade path tests.

func TestUpgradePathNonHAModeMTLSDisabled(t *testing.T) {
	// Ensure a clean environment.
	common.EnsureUninstall(false) // does not wait for pod deletion.
	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("setup v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			t.Run("delete CRDs "+p.previous.RuntimeVersion, common.DeleteCRD(p.previous.CustomResourceDefs))
			t.Run("delete CRDs "+p.next.RuntimeVersion, common.DeleteCRD(p.next.CustomResourceDefs))
		})
	}

	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			installOpts := common.TestOptions{
				HAEnabled:                false,
				MTLSEnabled:              false,
				ApplyComponentChanges:    true,
				ApplyHTTPEndpointChanges: false,
				CheckResourceExists: map[common.Resource]bool{
					common.CustomResourceDefs:  true,
					common.ClusterRoles:        true,
					common.ClusterRoleBindings: true,
				},
			}

			upgradeOpts := common.TestOptions{
				HAEnabled:   false,
				MTLSEnabled: false,
				// do not apply changes on upgrade, verify existing components and httpendpoints.
				ApplyComponentChanges:    false,
				ApplyHTTPEndpointChanges: false,
				CheckResourceExists: map[common.Resource]bool{
					common.CustomResourceDefs:  true,
					common.ClusterRoles:        true,
					common.ClusterRoleBindings: true,
				},
			}
			tests := getTestsOnUpgrade(p, installOpts, upgradeOpts)

			for _, tc := range tests {
				t.Run(tc.Name, tc.Callable)
			}
		})
	}
}

func TestUpgradePathNonHAModeMTLSEnabled(t *testing.T) {
	// Ensure a clean environment.
	common.EnsureUninstall(false) // does not wait for pod deletion.
	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("setup v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			t.Run("delete CRDs "+p.previous.RuntimeVersion, common.DeleteCRD(p.previous.CustomResourceDefs))
			t.Run("delete CRDs "+p.next.RuntimeVersion, common.DeleteCRD(p.next.CustomResourceDefs))
		})
	}

	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			installOpts := common.TestOptions{
				HAEnabled:                false,
				MTLSEnabled:              true,
				ApplyComponentChanges:    true,
				ApplyHTTPEndpointChanges: false,
				CheckResourceExists: map[common.Resource]bool{
					common.CustomResourceDefs:  true,
					common.ClusterRoles:        true,
					common.ClusterRoleBindings: true,
				},
			}

			upgradeOpts := common.TestOptions{
				HAEnabled:   false,
				MTLSEnabled: true,
				// do not apply changes on upgrade, verify existing components and httpendpoints.
				ApplyComponentChanges:    false,
				ApplyHTTPEndpointChanges: false,
				CheckResourceExists: map[common.Resource]bool{
					common.CustomResourceDefs:  true,
					common.ClusterRoles:        true,
					common.ClusterRoleBindings: true,
				},
			}
			tests := getTestsOnUpgrade(p, installOpts, upgradeOpts)

			for _, tc := range tests {
				t.Run(tc.Name, tc.Callable)
			}
		})
	}
}

func TestUpgradePathHAModeMTLSDisabled(t *testing.T) {
	// Ensure a clean environment.
	common.EnsureUninstall(false) // does not wait for pod deletion.
	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("setup v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			t.Run("delete CRDs "+p.previous.RuntimeVersion, common.DeleteCRD(p.previous.CustomResourceDefs))
			t.Run("delete CRDs "+p.next.RuntimeVersion, common.DeleteCRD(p.next.CustomResourceDefs))
		})
	}

	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			installOpts := common.TestOptions{
				HAEnabled:                true,
				MTLSEnabled:              false,
				ApplyComponentChanges:    true,
				ApplyHTTPEndpointChanges: false,
				CheckResourceExists: map[common.Resource]bool{
					common.CustomResourceDefs:  true,
					common.ClusterRoles:        true,
					common.ClusterRoleBindings: true,
				},
			}

			upgradeOpts := common.TestOptions{
				HAEnabled:   true,
				MTLSEnabled: false,
				// do not apply changes on upgrade, verify existing components and httpendpoints.
				ApplyComponentChanges:    false,
				ApplyHTTPEndpointChanges: false,
				CheckResourceExists: map[common.Resource]bool{
					common.CustomResourceDefs:  true,
					common.ClusterRoles:        true,
					common.ClusterRoleBindings: true,
				},
			}
			tests := getTestsOnUpgrade(p, installOpts, upgradeOpts)

			for _, tc := range tests {
				t.Run(tc.Name, tc.Callable)
			}
		})
	}
}

func TestUpgradePathHAModeMTLSEnabled(t *testing.T) {
	// Ensure a clean environment.
	common.EnsureUninstall(false) // does not wait for pod deletion.
	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("setup v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			t.Run("delete CRDs "+p.previous.RuntimeVersion, common.DeleteCRD(p.previous.CustomResourceDefs))
			t.Run("delete CRDs "+p.next.RuntimeVersion, common.DeleteCRD(p.next.CustomResourceDefs))
		})
	}

	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			installOpts := common.TestOptions{
				HAEnabled:                true,
				MTLSEnabled:              true,
				ApplyComponentChanges:    true,
				ApplyHTTPEndpointChanges: false,
				CheckResourceExists: map[common.Resource]bool{
					common.CustomResourceDefs:  true,
					common.ClusterRoles:        true,
					common.ClusterRoleBindings: true,
				},
			}

			upgradeOpts := common.TestOptions{
				HAEnabled:   true,
				MTLSEnabled: true,
				// do not apply changes on upgrade, verify existing components and httpendpoints.
				ApplyComponentChanges:    false,
				ApplyHTTPEndpointChanges: false,
				CheckResourceExists: map[common.Resource]bool{
					common.CustomResourceDefs:  true,
					common.ClusterRoles:        true,
					common.ClusterRoleBindings: true,
				},
			}
			tests := getTestsOnUpgrade(p, installOpts, upgradeOpts)

			for _, tc := range tests {
				t.Run(tc.Name, tc.Callable)
			}
		})
	}
}

// HTTPEndpoint Dapr resource is a new type as of v1.11.
// This test verifies install/upgrade functionality with this additional resource.
func TestUpgradeWithHTTPEndpoint(t *testing.T) {
	// Ensure a clean environment.
	common.EnsureUninstall(false) // does not wait for pod deletion.
	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("setup v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			t.Run("delete CRDs "+p.previous.RuntimeVersion, common.DeleteCRD(p.previous.CustomResourceDefs))
			t.Run("delete CRDs "+p.next.RuntimeVersion, common.DeleteCRD(p.next.CustomResourceDefs))
		})
	}

	for _, p := range supportedUpgradePaths {
		// only check runtime versions that support HTTPEndpoint resource.
		if !strings.Contains(p.next.RuntimeVersion, "1.11") {
			return
		}
		t.Run(fmt.Sprintf("v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			installOpts := common.TestOptions{
				HAEnabled:                true,
				MTLSEnabled:              true,
				ApplyComponentChanges:    false,
				ApplyHTTPEndpointChanges: true,
				CheckResourceExists: map[common.Resource]bool{
					common.CustomResourceDefs:  true,
					common.ClusterRoles:        true,
					common.ClusterRoleBindings: true,
				},
			}

			upgradeOpts := common.TestOptions{
				HAEnabled:   true,
				MTLSEnabled: true,
				// do not apply changes on upgrade, verify existing components and httpendpoints.
				ApplyComponentChanges:    false,
				ApplyHTTPEndpointChanges: true,
				CheckResourceExists: map[common.Resource]bool{
					common.CustomResourceDefs:  true,
					common.ClusterRoles:        true,
					common.ClusterRoleBindings: true,
				},
			}
			tests := getTestsOnUpgrade(p, installOpts, upgradeOpts)

			for _, tc := range tests {
				t.Run(tc.Name, tc.Callable)
			}
		})
	}
}
