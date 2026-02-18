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
	"testing"

	"github.com/Masterminds/semver/v3"

	"github.com/dapr/cli/tests/e2e/common"
)

const deleteCRDs = "delete CRDs "

type upgradePath struct {
	previous common.VersionDetails
	next     common.VersionDetails
}

const (
	latestRuntimeVersion         = "1.17.0-rc.8"
	latestRuntimeVersionMinusOne = "1.16.6"
	latestRuntimeVersionMinusTwo = "1.15.11"
	dashboardVersion             = "0.15.0"
)

var supportedUpgradePaths = []upgradePath{
	{
		// test upgrade on mariner images.
		previous: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersionMinusOne,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
		next: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersion,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
	},
	{
		previous: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersionMinusTwo,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
		next: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersionMinusOne,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
	},
	{
		previous: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersionMinusTwo,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
		next: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersion,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
	},
	// test downgrade.
	{
		previous: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersion,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
		next: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersionMinusOne,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
	},
	{
		previous: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersion,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
		next: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersionMinusTwo,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
	},
	{
		previous: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersionMinusOne,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		},
		next: common.VersionDetails{
			RuntimeVersion:      latestRuntimeVersionMinusTwo,
			DashboardVersion:    dashboardVersion,
			ClusterRoles:        []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"},
			ClusterRoleBindings: []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"},
			CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
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
		{Name: "previously applied http endpoints exist " + details.RuntimeVersion, Callable: common.HTTPEndpointsTestOnInstallUpgrade(installOpts, upgradeOpts)},
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
		common.TestCase{Name: deleteCRDs + p.previous.RuntimeVersion, Callable: common.DeleteCRD(p.previous.CustomResourceDefs)},
		common.TestCase{Name: deleteCRDs + p.next.RuntimeVersion, Callable: common.DeleteCRD(p.next.CustomResourceDefs)})

	return tests
}

// Upgrade path tests.

func TestUpgradePathNonHAModeMTLSDisabled(t *testing.T) {
	if common.ShouldSkipTest(common.DaprModeNonHA) {
		t.Skip(fmt.Sprintf("Skipping %s mode test", common.DaprModeNonHA))
	}

	// Ensure a clean environment.
	common.EnsureUninstall(false, false) // does not wait for pod deletion.

	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("setup v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			t.Run(deleteCRDs+p.previous.RuntimeVersion, common.DeleteCRD(p.previous.CustomResourceDefs))
			t.Run(deleteCRDs+p.next.RuntimeVersion, common.DeleteCRD(p.next.CustomResourceDefs))
		})

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
				TimeoutSeconds: 120,
			}
			tests := getTestsOnUpgrade(p, installOpts, upgradeOpts)

			for _, tc := range tests {
				t.Run(tc.Name, tc.Callable)
			}
		})
	}
}

func TestUpgradePathNonHAModeMTLSEnabled(t *testing.T) {
	if common.ShouldSkipTest(common.DaprModeNonHA) {
		t.Skip(fmt.Sprintf("Skipping %s mode test", common.DaprModeNonHA))
	}

	// Ensure a clean environment.
	common.EnsureUninstall(false, false) // does not wait for pod deletion.

	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("setup v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			t.Run(deleteCRDs+p.previous.RuntimeVersion, common.DeleteCRD(p.previous.CustomResourceDefs))
			t.Run(deleteCRDs+p.next.RuntimeVersion, common.DeleteCRD(p.next.CustomResourceDefs))
		})

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
				TimeoutSeconds: 120,
			}
			tests := getTestsOnUpgrade(p, installOpts, upgradeOpts)

			for _, tc := range tests {
				t.Run(tc.Name, tc.Callable)
			}
		})
	}
}

func TestUpgradePathHAModeMTLSDisabled(t *testing.T) {
	if common.ShouldSkipTest(common.DaprModeHA) {
		t.Skip(fmt.Sprintf("Skipping %s mode test", common.DaprModeHA))
	}

	// Ensure a clean environment.
	common.EnsureUninstall(false, false) // does not wait for pod deletion.

	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("setup v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			t.Run(deleteCRDs+p.previous.RuntimeVersion, common.DeleteCRD(p.previous.CustomResourceDefs))
			t.Run(deleteCRDs+p.next.RuntimeVersion, common.DeleteCRD(p.next.CustomResourceDefs))
		})
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
				TimeoutSeconds: 120,
			}
			tests := getTestsOnUpgrade(p, installOpts, upgradeOpts)

			for _, tc := range tests {
				t.Run(tc.Name, tc.Callable)
			}
		})
	}
}

func TestUpgradePathHAModeMTLSEnabled(t *testing.T) {
	if common.ShouldSkipTest(common.DaprModeHA) {
		t.Skip(fmt.Sprintf("Skipping %s mode test", common.DaprModeHA))
	}

	// Ensure a clean environment.
	common.EnsureUninstall(false, false) // does not wait for pod deletion.

	for _, p := range supportedUpgradePaths {
		t.Run(fmt.Sprintf("setup v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			t.Run(deleteCRDs+p.previous.RuntimeVersion, common.DeleteCRD(p.previous.CustomResourceDefs))
			t.Run(deleteCRDs+p.next.RuntimeVersion, common.DeleteCRD(p.next.CustomResourceDefs))
		})

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
				TimeoutSeconds: 120,
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
	if common.ShouldSkipTest(common.DaprModeHA) {
		t.Skip(fmt.Sprintf("Skipping %s mode test", common.DaprModeHA))
	}

	// Ensure a clean environment.
	common.EnsureUninstall(false, false) // does not wait for pod deletion.

	for _, p := range supportedUpgradePaths {
		ver, err := semver.NewVersion(p.previous.RuntimeVersion)
		if err != nil {
			t.Fatal(err)
		}

		// only check runtime versions that support HTTPEndpoint resource.
		if ver.Major() != 1 || ver.Minor() < 11 {
			return
		}

		t.Run(fmt.Sprintf("setup v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			t.Run(deleteCRDs+p.previous.RuntimeVersion, common.DeleteCRD(p.previous.CustomResourceDefs))
			t.Run(deleteCRDs+p.next.RuntimeVersion, common.DeleteCRD(p.next.CustomResourceDefs))
		})

		t.Run(fmt.Sprintf("v%s to v%s", p.previous.RuntimeVersion, p.next.RuntimeVersion), func(t *testing.T) {
			installOpts := common.TestOptions{
				HAEnabled:                true,
				MTLSEnabled:              true,
				ApplyComponentChanges:    true,
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
				TimeoutSeconds: 120,
			}
			tests := getTestsOnUpgrade(p, installOpts, upgradeOpts)

			for _, tc := range tests {
				t.Run(tc.Name, tc.Callable)
			}
		})
	}
}
