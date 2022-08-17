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
	"testing"

	"github.com/dapr/cli/tests/e2e/common"
)

var (
	currentRuntimeVersion   string
	currentDashboardVersion string
	currentVersionDetails   common.VersionDetails
)

// ensureCleanEnv function needs to be called in every Test function.
// sets necessary variable values and uninstalls any previously installed `dapr`.
func ensureCleanEnv(t *testing.T) {
	currentRuntimeVersion, currentDashboardVersion = common.GetVersionsFromEnv(t)

	currentVersionDetails = common.VersionDetails{
		RuntimeVersion:      currentRuntimeVersion,
		DashboardVersion:    currentDashboardVersion,
		CustomResourceDefs:  []string{"components.dapr.io", "configurations.dapr.io", "subscriptions.dapr.io"},
		ClusterRoles:        []string{"dapr-operator-admin", "dashboard-reader"},
		ClusterRoleBindings: []string{"dapr-operator", "dapr-role-tokenreview-binding", "dashboard-reader-global"},
		ImageVariant:        "",
	}
	// Ensure a clean environment
	common.EnsureUninstall(true) // does not wait for pod deletion
}

func TestKubernetesNonHAModeMTLSDisabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t)

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
	ensureCleanEnv(t)

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
	ensureCleanEnv(t)

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
	ensureCleanEnv(t)

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

// Test for certificate renewal

func TestRenewCertificateMTLSEnabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t)

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

	// tests for certifcate renewal with newly generated certificates.
	tests = append(tests, []common.TestCase{
		{"Renew certificate which expires in less than 30 days", common.GenerateNewCertAndRenew(currentVersionDetails, installOpts)},
	}...)
	tests = append(tests, common.GetTestsPostCertificateRenewal(currentVersionDetails, installOpts)...)
	tests = append(tests, []common.TestCase{
		{"Cert Expiry warning message check " + currentVersionDetails.RuntimeVersion, common.CheckMTLSStatus(currentVersionDetails, installOpts, true)},
	}...)

	// tests for certificate renewal with provided certificates.
	tests = append(tests, []common.TestCase{
		{"Renew certificate which expires in after 30 days", common.UseProvidedNewCertAndRenew(currentVersionDetails, installOpts)},
	}...)
	tests = append(tests, common.GetTestsPostCertificateRenewal(currentVersionDetails, installOpts)...)
	tests = append(tests, []common.TestCase{
		{"Cert Expiry no warning message check " + currentVersionDetails.RuntimeVersion, common.CheckMTLSStatus(currentVersionDetails, installOpts, false)},
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

func TestRenewCertificateMTLSDisabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t)

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

	// tests for certifcate renewal with newly generated certificates.
	tests = append(tests, []common.TestCase{
		{"Renew certificate which expires in less than 30 days", common.GenerateNewCertAndRenew(currentVersionDetails, installOpts)},
	}...)
	tests = append(tests, common.GetTestsPostCertificateRenewal(currentVersionDetails, installOpts)...)
	tests = append(tests, []common.TestCase{
		{"Cert Expiry warning message check " + currentVersionDetails.RuntimeVersion, common.CheckMTLSStatus(currentVersionDetails, installOpts, true)},
	}...)

	// tests for certificate renewal with provided certificates.
	tests = append(tests, []common.TestCase{
		{"Renew certificate which expires in after 30 days", common.UseProvidedNewCertAndRenew(currentVersionDetails, installOpts)},
	}...)
	tests = append(tests, common.GetTestsPostCertificateRenewal(currentVersionDetails, installOpts)...)
	tests = append(tests, []common.TestCase{
		{"Cert Expiry no warning message check " + currentVersionDetails.RuntimeVersion, common.CheckMTLSStatus(currentVersionDetails, installOpts, false)},
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

func TestRenewCertWithPrivateKey(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t)

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
	ensureCleanEnv(t)

	tests := []common.TestCase{}
	var installOpts = common.TestOptions{
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
	var installOpts = common.TestOptions{
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

func TestKubernetesInstallwithMarinerImages(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t)

	//	install with mariner images
	currentVersionDetails.ImageVariant = "mariner"

	tests := []common.TestCase{}
	var installOpts = common.TestOptions{
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
