//go:build e2e
// +build e2e

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
	currentRuntimeVersion     string
	currentDashboardVersion   string
	currentVersionDetails     common.VersionDetails
	clusterRoles1_9_X         = []string{"dapr-operator-admin", "dashboard-reader"}
	clusterRoleBindings1_9_X  = []string{"dapr-operator", "dapr-role-tokenreview-binding", "dashboard-reader-global"}
	clusterRoles1_10_X        = []string{"dapr-dashboard", "dapr-injector", "dapr-operator-admin", "dapr-placement", "dapr-sentry"}
	clusterRoleBindings1_10_X = []string{"dapr-operator-admin", "dapr-dashboard", "dapr-injector", "dapr-placement", "dapr-sentry"}
)

// ensureCleanEnv function needs to be called in every Test function.
// sets necessary variable values and uninstalls any previously installed `dapr`.
func ensureCleanEnv(t *testing.T, useDaprLatestVersion bool) {
	currentRuntimeVersion, currentDashboardVersion = common.GetVersionsFromEnv(t, useDaprLatestVersion)

	currentVersionDetails = common.VersionDetails{
		RuntimeVersion:       currentRuntimeVersion,
		DashboardVersion:     currentDashboardVersion,
		CustomResourceDefs:   []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io", "resiliencies.dapr.io", "httpendpoints.dapr.io"},
		ImageVariant:         "",
		UseDaprLatestVersion: useDaprLatestVersion,
	}
	if strings.HasPrefix(currentRuntimeVersion, "1.9.") {
		currentVersionDetails.ClusterRoles = clusterRoles1_9_X
		currentVersionDetails.ClusterRoleBindings = clusterRoleBindings1_9_X
	} else {
		currentVersionDetails.ClusterRoles = clusterRoles1_10_X
		currentVersionDetails.ClusterRoleBindings = clusterRoleBindings1_10_X
	}
	// Ensure a clean environment
	common.EnsureUninstall(true) // does not wait for pod deletion
}

func TestKubernetesNonHAModeMTLSDisabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           false,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesHAModeMTLSDisabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             true,
		MTLSEnabled:           false,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesNonHAModeMTLSEnabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesHAModeMTLSEnabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             true,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		UninstallAll: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  false,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesInitWithCustomCert(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		InitWithCustomCert:    true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

// Test for certificate renewal

func TestRenewCertificateMTLSEnabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// tests for certifcate renewal.
	tests = append(tests, common.GetTestForCertRenewal(currentVersionDetails, installOpts)...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestRenewCertificateMTLSDisabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           false,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// tests for certifcate renewal.
	tests = append(tests, common.GetTestForCertRenewal(currentVersionDetails, installOpts)...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestRenewCertWithPrivateKey(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// tests for certifcate renewal with newly generated certificates when pem encoded private root.key file is provided
	tests = append(tests, []common.TestCase{
		{"Renew certificate which expires in less than 30 days", common.UseProvidedPrivateKeyAndRenewCerts(currentVersionDetails, installOpts)},
	}...)

	tests = append(tests, common.GetTestsPostCertificateRenewal(currentVersionDetails, installOpts)...)
	tests = append(tests, []common.TestCase{
		{"Cert Expiry warning message check " + currentVersionDetails.RuntimeVersion, common.CheckMTLSStatus(currentVersionDetails, installOpts, true)},
	}...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesUninstall(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)
	// setup tests
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestRenewCertWithIncorrectFlags(t *testing.T) {
	common.EnsureUninstall(true)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// tests for certifcate renewal with incorrect set of flags provided.
	tests = append(tests, []common.TestCase{
		{"Renew certificate with incorrect flags", common.NegativeScenarioForCertRenew()},
	}...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

// install dapr control plane with mariner docker images.
// Renew the certificate of this control plane.
func TestK8sInstallwithMarinerImagesAndRenewCertificate(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	//	install with mariner images
	currentVersionDetails.ImageVariant = "mariner"

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// tests for certifcate renewal.
	tests = append(tests, common.GetTestForCertRenewal(currentVersionDetails, installOpts)...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesInstallwithoutRuntimeVersionFlag(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, true)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestK8sInstallwithoutRuntimeVersionwithMarinerImagesFlag(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, true)

	//	install with mariner images
	currentVersionDetails.ImageVariant = "mariner"

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}
