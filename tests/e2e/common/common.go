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

package common

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	core_v1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/tests/e2e/spawn"

	k8s "k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Resource int

const (
	DaprTestNamespace           = "dapr-cli-tests"
	CustomResourceDefs Resource = iota
	ClusterRoles
	ClusterRoleBindings

	numHAPods    = 13
	numNonHAPods = 5
)

type VersionDetails struct {
	RuntimeVersion       string
	DashboardVersion     string
	ImageVariant         string
	CustomResourceDefs   []string
	ClusterRoles         []string
	ClusterRoleBindings  []string
	UseDaprLatestVersion bool
}

type TestOptions struct {
	HAEnabled                bool
	MTLSEnabled              bool
	ApplyComponentChanges    bool
	ApplyHTTPEndpointChanges bool
	CheckResourceExists      map[Resource]bool
	UninstallAll             bool
	InitWithCustomCert       bool
}

type TestCase struct {
	Name     string
	Callable func(*testing.T)
}

// GetVersionsFromEnv will return values from required environment variables.
// parameter `latest` is used to determine if the latest versions of dapr & dashboard should be used.
// if environment variables are not set it fails the test.
func GetVersionsFromEnv(t *testing.T, latest bool) (string, string) {
	var daprRuntimeVersion, daprDashboardVersion string
	runtimeEnvVar := "DAPR_RUNTIME_PINNED_VERSION"
	dashboardEnvVar := "DAPR_DASHBOARD_PINNED_VERSION"
	if latest {
		runtimeEnvVar = "DAPR_RUNTIME_LATEST_STABLE_VERSION"
		dashboardEnvVar = "DAPR_DASHBOARD_LATEST_STABLE_VERSION"
	}
	if runtimeVersion, ok := os.LookupEnv(runtimeEnvVar); ok {
		daprRuntimeVersion = runtimeVersion
	} else {
		t.Fatalf("env var \"%s\" not set", runtimeEnvVar)
	}
	if dashboardVersion, ok := os.LookupEnv(dashboardEnvVar); ok {
		daprDashboardVersion = dashboardVersion
	} else {
		t.Fatalf("env var \"%s\" not set", dashboardEnvVar)
	}
	return daprRuntimeVersion, daprDashboardVersion
}

func UpgradeTest(details VersionDetails, opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		args := []string{
			"upgrade", "-k",
			"--runtime-version", details.RuntimeVersion,
			"--log-as-json",
		}

		hasDashboardInDaprChart, err := kubernetes.IsDashboardIncluded(details.RuntimeVersion)
		require.NoError(t, err, "failed to check if dashboard is included in dapr chart")

		if !hasDashboardInDaprChart {
			args = append(args, "--dashboard-version", details.DashboardVersion)
		}

		if details.ImageVariant != "" {
			args = append(args, "--image-variant", details.ImageVariant)
		}

		output, err := spawn.Command(daprPath, args...)
		t.Log(output)
		require.NoError(t, err, "upgrade failed")

		done := make(chan struct{})
		podsRunning := make(chan struct{})

		go waitAllPodsRunning(t, DaprTestNamespace, opts.HAEnabled, done, podsRunning)
		select {
		case <-podsRunning:
			t.Logf("verified all pods running in namespace %s are running after upgrade", DaprTestNamespace)
		case <-time.After(2 * time.Minute):
			done <- struct{}{}
			t.Logf("timeout verifying all pods running in namespace %s", DaprTestNamespace)
			t.FailNow()
		}
		validatePodsOnInstallUpgrade(t, details)
	}
}

func EnsureUninstall(all bool) (string, error) {
	daprPath := GetDaprPath()

	var _command [10]string
	command := append(_command[0:], "uninstall", "-k")

	if all {
		command = append(command, "--all")
	}

	command = append(command,
		"-n", DaprTestNamespace,
		"--log-as-json")

	return spawn.Command(daprPath, command...)
}

func DeleteCRD(crds []string) func(*testing.T) {
	return func(t *testing.T) {
		for _, crd := range crds {
			output, err := spawn.Command("kubectl", "delete", "crd", crd)
			if err != nil {
				// CRD already deleted and not found.
				require.Contains(t, output, "Error from server (NotFound)")
				continue
			} else {
				require.NoErrorf(t, err, "expected no error on deleting crd %s", crd)
			}
			require.Equal(t, fmt.Sprintf("customresourcedefinition.apiextensions.k8s.io \"%s\" deleted\n", crd), output, "expected output to match")
		}
	}
}

// Get Test Cases.

func GetTestsOnInstall(details VersionDetails, opts TestOptions) []TestCase {
	return []TestCase{
		{"install " + details.RuntimeVersion, installTest(details, opts)},
		{"crds exist " + details.RuntimeVersion, CRDTest(details, opts)},
		{"clusterroles exist " + details.RuntimeVersion, ClusterRolesTest(details, opts)},
		{"clusterrolebindings exist " + details.RuntimeVersion, ClusterRoleBindingsTest(details, opts)},
		{"apply and check components exist " + details.RuntimeVersion, ComponentsTestOnInstallUpgrade(opts)},
		{"apply and check httpendpoints exist " + details.RuntimeVersion, HTTPEndpointsTestOnInstallUpgrade(opts)},
		{"check mtls " + details.RuntimeVersion, MTLSTestOnInstallUpgrade(opts)},
		{"status check " + details.RuntimeVersion, StatusTestOnInstallUpgrade(details, opts)},
	}
}

func GetTestsOnUninstall(details VersionDetails, opts TestOptions) []TestCase {
	return []TestCase{
		{"uninstall " + details.RuntimeVersion, uninstallTest(opts.UninstallAll)}, // waits for pod deletion.
		{"cluster not exist", kubernetesTestOnUninstall()},
		{"crds exist on uninstall " + details.RuntimeVersion, CRDTest(details, opts)},
		{"clusterroles not exist " + details.RuntimeVersion, ClusterRolesTest(details, opts)},
		{"clusterrolebindings not exist " + details.RuntimeVersion, ClusterRoleBindingsTest(details, opts)},
		{"check components exist on uninstall " + details.RuntimeVersion, componentsTestOnUninstall(opts.UninstallAll)},
		{"check httpendpoints exist on uninstall " + details.RuntimeVersion, httpEndpointsTestOnUninstall(opts)},
		{"check mtls error " + details.RuntimeVersion, uninstallMTLSTest()},
		{"check status error " + details.RuntimeVersion, statusTestOnUninstall()},
	}
}

func GetTestForCertRenewal(currentVersionDetails VersionDetails, installOpts TestOptions) []TestCase {
	tests := []TestCase{}
	tests = append(tests, []TestCase{
		{"Renew certificate which expires in less than 30 days", GenerateNewCertAndRenew(currentVersionDetails, installOpts)},
	}...)
	tests = append(tests, GetTestsPostCertificateRenewal(currentVersionDetails, installOpts)...)
	tests = append(tests, []TestCase{
		{"Cert Expiry warning message check " + currentVersionDetails.RuntimeVersion, CheckMTLSStatus(currentVersionDetails, installOpts, true)},
	}...)

	// tests for certificate renewal with provided certificates.
	tests = append(tests, []TestCase{
		{"Renew certificate which expires in after 30 days", UseProvidedNewCertAndRenew(currentVersionDetails, installOpts)},
	}...)
	tests = append(tests, GetTestsPostCertificateRenewal(currentVersionDetails, installOpts)...)
	tests = append(tests, []TestCase{
		{"Cert Expiry no warning message check " + currentVersionDetails.RuntimeVersion, CheckMTLSStatus(currentVersionDetails, installOpts, false)},
	}...)
	return tests
}

func GetTestsPostCertificateRenewal(details VersionDetails, opts TestOptions) []TestCase {
	return []TestCase{
		{"crds exist " + details.RuntimeVersion, CRDTest(details, opts)},
		{"clusterroles exist " + details.RuntimeVersion, ClusterRolesTest(details, opts)},
		{"clusterrolebindings exist " + details.RuntimeVersion, ClusterRoleBindingsTest(details, opts)},
		{"status check " + details.RuntimeVersion, StatusTestOnInstallUpgrade(details, opts)},
	}
}

func MTLSTestOnInstallUpgrade(opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		output, err := spawn.Command(daprPath, "mtls", "-k")
		require.NoError(t, err, "expected no error on querying for mtls")
		if !opts.MTLSEnabled {
			t.Log("check mtls disabled")
			require.Contains(t, output, "Mutual TLS is disabled in your Kubernetes cluster", "expected output to match")
		} else {
			t.Log("check mtls enabled")
			require.Contains(t, output, "Mutual TLS is enabled in your Kubernetes cluster", "expected output to match")
		}

		// expiry.
		output, err = spawn.Command(daprPath, "mtls", "expiry")
		require.NoError(t, err, "expected no error on querying for mtls expiry")
		assert.Contains(t, output, "Root certificate expires in", "expected output to contain string")
		assert.Contains(t, output, "Expiry date:", "expected output to contain string")
		if opts.InitWithCustomCert {
			t.Log("check mtls expiry with custom cert: ", output)
		}

		// export
		// check that the dir does not exist now.
		_, err = os.Stat("./certs")
		if assert.Error(t, err) {
			assert.True(t, os.IsNotExist(err), err.Error())
		}

		output, err = spawn.Command(daprPath, "mtls", "export", "-o", "./certs")
		require.NoError(t, err, "expected no error on mtls export")
		require.Contains(t, output, "Trust certs successfully exported to", "expected output to contain string")

		// check export success.
		_, err = os.Stat("./certs")
		require.NoError(t, err, "expected directory to exist")
		_, err = os.Stat("./certs/ca.crt")
		require.NoError(t, err, "expected file to exist")
		_, err = os.Stat("./certs/issuer.crt")
		require.NoError(t, err, "expected file to exist")
		_, err = os.Stat("./certs/issuer.key")
		require.NoError(t, err, "expected file to exist")
		err = os.RemoveAll("./certs")
		if err != nil {
			t.Logf("error on removing local certs directory %s", err.Error())
		}
	}
}

func ComponentsTestOnInstallUpgrade(opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		// if dapr is installed.
		if opts.ApplyComponentChanges {
			// apply any changes to the component.
			t.Log("apply component changes")
			output, err := spawn.Command("kubectl", "apply", "-f", "../testdata/namespace.yaml")
			t.Log(output)
			require.NoError(t, err, "expected no error on kubectl apply")
			output, err = spawn.Command("kubectl", "apply", "-f", "../testdata/statestore.yaml")
			t.Log(output)
			require.NoError(t, err, "expected no error on kubectl apply")
			require.Equal(t, "component.dapr.io/statestore created\ncomponent.dapr.io/statestore created\n", output, "expceted output to match")
		}

		t.Log("check applied component exists")
		output, err := spawn.Command(daprPath, "components", "-k")
		require.NoError(t, err, "expected no error on calling dapr components")
		componentOutputCheck(t, output, false)
	}
}

func HTTPEndpointsTestOnInstallUpgrade(opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		// if dapr is installed with httpendpoints.
		if opts.ApplyHTTPEndpointChanges {
			// apply any changes to the httpendpoint.
			t.Log("apply httpendpoint changes")
			output, err := spawn.Command("kubectl", "apply", "-f", "../testdata/namespace.yaml")
			t.Log(output)
			require.NoError(t, err, "expected no error on kubectl apply")
			output, err = spawn.Command("kubectl", "apply", "-f", "../testdata/httpendpoint.yaml")
			t.Log(output)
			require.NoError(t, err, "expected no error on kubectl apply")
			require.Equal(t, "httpendpoints.dapr.io/httpendpoint created\nhttpendpoints.dapr.io/httpendpoint created\n", output, "expected output to match")
			httpEndpointOutputCheck(t, output)

			t.Log("check applied httpendpoint exists")
			_, err = spawn.Command("kubectl", "get", "httpendpoint")
			require.NoError(t, err, "expected no error on calling to retrieve httpendpoints")
		}
	}
}

func StatusTestOnInstallUpgrade(details VersionDetails, opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		output, err := spawn.Command(daprPath, "status", "-k")
		require.NoError(t, err, "status check failed")
		var notFound map[string][]string
		if !opts.HAEnabled {
			notFound = map[string][]string{
				"dapr-sentry":           {details.RuntimeVersion, "1"},
				"dapr-sidecar-injector": {details.RuntimeVersion, "1"},
				"dapr-dashboard":        {details.DashboardVersion, "1"},
				"dapr-placement-server": {details.RuntimeVersion, "1"},
				"dapr-operator":         {details.RuntimeVersion, "1"},
			}
		} else {
			notFound = map[string][]string{
				"dapr-sentry":           {details.RuntimeVersion, "3"},
				"dapr-sidecar-injector": {details.RuntimeVersion, "3"},
				"dapr-dashboard":        {details.DashboardVersion, "1"},
				"dapr-placement-server": {details.RuntimeVersion, "3"},
				"dapr-operator":         {details.RuntimeVersion, "3"},
			}
		}

		if details.ImageVariant != "" {
			notFound["dapr-sentry"][0] = notFound["dapr-sentry"][0] + "-" + details.ImageVariant
			notFound["dapr-sidecar-injector"][0] = notFound["dapr-sidecar-injector"][0] + "-" + details.ImageVariant
			notFound["dapr-placement-server"][0] = notFound["dapr-placement-server"][0] + "-" + details.ImageVariant
			notFound["dapr-operator"][0] = notFound["dapr-operator"][0] + "-" + details.ImageVariant
		}

		lines := strings.Split(output, "\n")[1:] // remove header of status.
		t.Logf("dapr status -k infos: \n%s\n", lines)
		for _, line := range lines {
			cols := strings.Fields(strings.TrimSpace(line))
			if len(cols) > 6 { // atleast 6 fields are verified from status (Age and created time are not).
				if toVerify, ok := notFound[cols[0]]; ok { // get by name.
					require.Equal(t, DaprTestNamespace, cols[1], "namespace must match")
					require.Equal(t, "True", cols[2], "healthly field must be true")
					require.Equal(t, "Running", cols[3], "pods must be Running")
					require.Equal(t, toVerify[1], cols[4], "replicas must be equal")
					// TODO: Skip the dashboard version check for now until the helm chart is updated.
					if cols[0] != "dapr-dashboard" {
						require.Equal(t, toVerify[0], cols[5], "versions must match")
					}
					delete(notFound, cols[0])
				}
			}
		}
		assert.Empty(t, notFound)
	}
}

func ClusterRoleBindingsTest(details VersionDetails, opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		foundMap := details.constructFoundMap(ClusterRoleBindings)
		wanted, ok := opts.CheckResourceExists[ClusterRoleBindings]
		if !ok {
			t.Errorf("check on cluster roles bindings called when not defined in test options")
		}

		ctx := context.Background()
		k8sClient, err := getClient()
		require.NoError(t, err)

		var listContinue string
		for {
			list, err := k8sClient.
				RbacV1().
				ClusterRoleBindings().
				List(ctx, v1.ListOptions{
					Limit:    100,
					Continue: listContinue,
				})
			require.NoError(t, err)

			for _, roleBinding := range list.Items {
				if _, exists := foundMap[roleBinding.Name]; exists {
					foundMap[roleBinding.Name] = true
				}
			}

			listContinue = list.Continue
			if listContinue == "" {
				break
			}
		}

		for name, found := range foundMap {
			assert.Equal(t, wanted, found, "cluster role binding %s, found = %t, wanted = %t", name, found, wanted)
		}
	}
}

func ClusterRolesTest(details VersionDetails, opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		foundMap := details.constructFoundMap(ClusterRoles)
		wanted, ok := opts.CheckResourceExists[ClusterRoles]
		if !ok {
			t.Errorf("check on cluster roles called when not defined in test options")
		}
		ctx := context.Background()
		k8sClient, err := getClient()
		require.NoError(t, err)

		var listContinue string
		for {
			list, err := k8sClient.RbacV1().ClusterRoles().List(ctx, v1.ListOptions{
				Limit:    100,
				Continue: listContinue,
			})
			require.NoError(t, err)

			for _, role := range list.Items {
				if _, exists := foundMap[role.Name]; exists {
					foundMap[role.Name] = true
				}
			}

			listContinue = list.Continue
			if listContinue == "" {
				break
			}
		}

		for name, found := range foundMap {
			assert.Equal(t, wanted, found, "cluster role %s, found = %t, wanted = %t", name, found, wanted)
		}
	}
}

func CRDTest(details VersionDetails, opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		foundMap := details.constructFoundMap(CustomResourceDefs)
		wanted, ok := opts.CheckResourceExists[CustomResourceDefs]
		if !ok {
			t.Errorf("check on CRDs called when not defined in test options")
		}
		ctx := context.Background()
		cfg, err := getConfig()
		require.NoError(t, err)

		apiextensionsClientSet, err := apiextensionsclient.NewForConfig(cfg)
		require.NoError(t, err)

		var listContinue string
		for {
			list, err := apiextensionsClientSet.
				ApiextensionsV1().
				CustomResourceDefinitions().
				List(ctx, v1.ListOptions{
					Limit:    100,
					Continue: listContinue,
				})
			require.NoError(t, err)

			for _, crd := range list.Items {
				if _, exists := foundMap[crd.Name]; exists {
					foundMap[crd.Name] = true
				}
			}

			listContinue = list.Continue
			if listContinue == "" {
				break
			}
		}

		for name, found := range foundMap {
			assert.Equal(t, wanted, found, "cluster role binding %s, found = %t, wanted = %t", name, found, wanted)
		}
	}
}

func GenerateNewCertAndRenew(details VersionDetails, opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		err := exportCurrentCertificate(daprPath)
		require.NoError(t, err, "expected no error on certificate exporting")

		args := []string{
			"mtls",
			"renew-certificate",
			"-k",
			"--valid-until", "20",
			"--restart",
		}
		if details.ImageVariant != "" {
			args = append(args, "--image-variant", details.ImageVariant)
		}
		output, err := spawn.Command(daprPath, args...)
		t.Log(output)
		require.NoError(t, err, "expected no error on certificate renewal")

		done := make(chan struct{})
		podsRunning := make(chan struct{})

		go waitAllPodsRunning(t, DaprTestNamespace, opts.HAEnabled, done, podsRunning)
		select {
		case <-podsRunning:
			t.Logf("verified all pods running in namespace %s are running after certficate change", DaprTestNamespace)
		case <-time.After(2 * time.Minute):
			done <- struct{}{}
			t.Logf("timeout verifying all pods running in namespace %s", DaprTestNamespace)
			t.FailNow()
		}
		assert.Contains(t, output, "Certificate rotation is successful!")
	}
}

func UseProvidedPrivateKeyAndRenewCerts(details VersionDetails, opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		args := []string{
			"mtls", "renew-certificate", "-k",
			"--private-key", "../testdata/example-root.key",
			"--valid-until", "20",
		}
		if details.ImageVariant != "" {
			args = append(args, "--image-variant", details.ImageVariant)
		}
		output, err := spawn.Command(daprPath, args...)
		t.Log(output)
		require.NoError(t, err, "expected no error on certificate renewal")

		done := make(chan struct{})
		podsRunning := make(chan struct{})

		go waitAllPodsRunning(t, DaprTestNamespace, opts.HAEnabled, done, podsRunning)
		select {
		case <-podsRunning:
			t.Logf("verified all pods running in namespace %s are running after certficate change", DaprTestNamespace)
		case <-time.After(2 * time.Minute):
			done <- struct{}{}
			t.Logf("timeout verifying all pods running in namespace %s", DaprTestNamespace)
			t.FailNow()
		}
		assert.Contains(t, output, "Certificate rotation is successful!")
	}
}

func UseProvidedNewCertAndRenew(details VersionDetails, opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		args := []string{
			"mtls", "renew-certificate", "-k",
			"--ca-root-certificate", "./certs/ca.crt",
			"--issuer-private-key", "./certs/issuer.key",
			"--issuer-public-certificate", "./certs/issuer.crt",
			"--restart",
		}
		if details.ImageVariant != "" {
			args = append(args, "--image-variant", details.ImageVariant)
		}
		output, err := spawn.Command(daprPath, args...)
		t.Log(output)
		require.NoError(t, err, "expected no error on certificate renewal")

		done := make(chan struct{})
		podsRunning := make(chan struct{})

		go waitAllPodsRunning(t, DaprTestNamespace, opts.HAEnabled, done, podsRunning)
		select {
		case <-podsRunning:
			t.Logf("verified all pods running in namespace %s are running after certficate change", DaprTestNamespace)
		case <-time.After(2 * time.Minute):
			done <- struct{}{}
			t.Logf("timeout verifying all pods running in namespace %s", DaprTestNamespace)
			t.FailNow()
		}

		assert.Contains(t, output, "Certificate rotation is successful!")

		// remove cert directory created earlier.
		os.RemoveAll("./certs")
	}
}

func NegativeScenarioForCertRenew() func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		args := []string{
			"mtls", "renew-certificate", "-k",
			"--ca-root-certificate", "invalid_cert_file.pem",
		}
		output, err := spawn.Command(daprPath, args...)
		t.Log(output)
		require.Error(t, err, "expected error on certificate renewal")
		assert.Contains(t, output, "certificate rotation failed: all required flags for this certificate rotation path")

		args = []string{
			"mtls", "renew-certificate", "-k",
			"--ca-root-certificate", "invalid_cert_file.pem",
			"--issuer-private-key", "invalid_cert_key.pem",
			"--issuer-public-certificate", "invalid_cert_file.pem",
		}
		output, err = spawn.Command(daprPath, args...)
		t.Log(output)
		require.Error(t, err, "expected error on certificate renewal")
		assert.Contains(t, output, "certificate rotation failed: open invalid_cert_file.pem: no such file or directory")

		args = []string{
			"mtls", "renew-certificate", "-k",
			"--ca-root-certificate", "invalid_cert_file.pem",
			"--private-key", "invalid_root_key.pem",
		}
		output, err = spawn.Command(daprPath, args...)
		t.Log(output)
		require.Error(t, err, "expected error on certificate renewal")
		assert.Contains(t, output, "certificate rotation failed: all required flags for this certificate rotation path")

		args = []string{
			"mtls", "renew-certificate", "-k",
			"--private-key", "invalid_root_key.pem",
		}
		output, err = spawn.Command(daprPath, args...)
		t.Log(output)
		require.Error(t, err, "expected error on certificate renewal")
		assert.Contains(t, output, "certificate rotation failed: open invalid_root_key.pem: no such file or directory")
	}
}

func CheckMTLSStatus(details VersionDetails, opts TestOptions, shouldWarningExist bool) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		output, err := spawn.Command(daprPath, "mtls", "-k")
		require.NoError(t, err, "expected no error on querying for mtls")
		if !opts.MTLSEnabled {
			t.Log("check mtls disabled")
			require.Contains(t, output, "Mutual TLS is disabled in your Kubernetes cluster", "expected output to match")
		} else {
			t.Log("check mtls enabled")
			require.Contains(t, output, "Mutual TLS is enabled in your Kubernetes cluster", "expected output to match")
		}
		output, err = spawn.Command(daprPath, "status", "-k")
		require.NoError(t, err, "status check failed")
		if shouldWarningExist {
			assert.Contains(t, output, "Dapr root certificate of your Kubernetes cluster expires in", "expected output to contain string")
			assert.Contains(t, output, "Expiry date:", "expected output to contain string")
			assert.Contains(t, output, "Please see docs.dapr.io for certificate renewal instructions to avoid service interruptions")
		} else {
			assert.NotContains(t, output, "Dapr root certificate of your Kubernetes cluster expires in", "expected output to contain string")
			assert.NotContains(t, output, "Expiry date:", "expected output to contain string")
			assert.NotContains(t, output, "Please see docs.dapr.io for certificate renewal instructions to avoid service interruptions")
		}
	}
}

// Unexported functions.

func (v VersionDetails) constructFoundMap(res Resource) map[string]bool {
	foundMap := map[string]bool{}
	var list []string
	switch res {
	case CustomResourceDefs:
		list = v.CustomResourceDefs
	case ClusterRoles:
		list = v.ClusterRoles
	case ClusterRoleBindings:
		list = v.ClusterRoleBindings
	}

	for _, val := range list {
		foundMap[val] = false
	}
	return foundMap
}

func GetDaprPath() string {
	distDir := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	return filepath.Join("..", "..", "..", "dist", distDir, "release", "dapr")
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows.
}

func getConfig() (*rest.Config, error) {
	var kubeconfig string
	if home := homeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	kubeConfigEnv := os.Getenv("KUBECONFIG")

	if len(kubeConfigEnv) != 0 {
		kubeConfigs := strings.Split(kubeConfigEnv, ":")
		if len(kubeConfigs) > 1 {
			return nil, fmt.Errorf("multiple kubeconfigs in KUBECONFIG environment variable - %s", kubeConfigEnv)
		}
		kubeconfig = kubeConfigs[0]
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// getClient returns a new Kubernetes client.
func getClient() (*k8s.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	return k8s.NewForConfig(config)
}

func installTest(details VersionDetails, opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		args := []string{
			"init", "-k",
			"--wait",
			"-n", DaprTestNamespace,
			"--log-as-json",
		}
		if !details.UseDaprLatestVersion {
			// TODO: Pass dashboard-version also when charts are released.
			args = append(args, "--runtime-version", details.RuntimeVersion)
		}
		if opts.HAEnabled {
			args = append(args, "--enable-ha")
		}
		if !opts.MTLSEnabled {
			t.Log("install without mtls")
			args = append(args, "--enable-mtls=false")
		} else {
			t.Log("install with mtls")
		}
		if details.ImageVariant != "" {
			args = append(args, "--image-variant", details.ImageVariant)
		}
		if opts.InitWithCustomCert {
			certParam := []string{
				"--ca-root-certificate", "../testdata/customcerts/root.pem",
				"--issuer-private-key", "../testdata/customcerts/issuer.key",
				"--issuer-public-certificate", "../testdata/customcerts/issuer.pem",
			}
			args = append(args, certParam...)
		}
		output, err := spawn.Command(daprPath, args...)
		t.Log(output)
		require.NoError(t, err, "init failed")

		validatePodsOnInstallUpgrade(t, details)
	}
}

func uninstallTest(all bool) func(t *testing.T) {
	return func(t *testing.T) {
		output, err := EnsureUninstall(all)
		t.Log(output)
		require.NoError(t, err, "uninstall failed")
		// wait for pods to be deleted completely.
		// needed to verify status checks fails correctly.
		podsDeleted := make(chan struct{})
		done := make(chan struct{})
		t.Log("waiting for pods to be deleted completely")
		go waitPodDeletion(t, done, podsDeleted)
		select {
		case <-podsDeleted:
			t.Log("pods were deleted as expected on uninstall")
			return
		case <-time.After(2 * time.Minute):
			done <- struct{}{}
			t.Error("timeout verifying pods were deleted as expectedx")
		}
	}
}

func kubernetesTestOnUninstall() func(t *testing.T) {
	return func(t *testing.T) {
		_, err := EnsureUninstall(true)
		require.NoError(t, err, "uninstall failed")
		daprPath := GetDaprPath()
		output, err := spawn.Command(daprPath, "uninstall", "-k")
		require.NoError(t, err, "expected no error on uninstall without install")
		require.Contains(t, output, "Removing Dapr from your cluster...", "expected output to contain message")
		require.Contains(t, output, "WARNING: dapr release does not exist", "expected output to contain message")
		require.Contains(t, output, "Dapr has been removed successfully")
	}
}

func uninstallMTLSTest() func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		output, err := spawn.Command(daprPath, "mtls", "-k")
		require.Error(t, err, "expected error to be return if dapr not installed")
		require.Contains(t, output, "error checking mTLS: system configuration not found", "expected output to match")
	}
}

func componentsTestOnUninstall(all bool) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		// On Dapr uninstall CRDs are not removed, consequently the components will not be removed.
		// TODO: Related to https://github.com/dapr/cli/issues/656.
		// For now the components remain.
		output, err := spawn.Command(daprPath, "components", "-k")
		require.NoError(t, err, "expected no error on calling dapr components")
		componentOutputCheck(t, output, all)

		// If --all, then the below does not need to run.
		if all {
			output, err = spawn.Command("kubectl", "delete", "-f", "../testdata/namespace.yaml")
			require.NoError(t, err, "expected no error on kubectl delete")
			t.Log(output)
			return
		}

		// Manually remove components and verify output.
		output, err = spawn.Command("kubectl", "delete", "-f", "../testdata/statestore.yaml")
		require.NoError(t, err, "expected no error on kubectl apply")
		require.Equal(t, "component.dapr.io \"statestore\" deleted\ncomponent.dapr.io \"statestore\" deleted\n", output, "expected output to match")
		output, err = spawn.Command("kubectl", "delete", "-f", "../testdata/namespace.yaml")
		require.NoError(t, err, "expected no error on kubectl delete")
		t.Log(output)
		output, err = spawn.Command(daprPath, "components", "-k")
		require.NoError(t, err, "expected no error on calling dapr components")
		lines := strings.Split(output, "\n")

		// An extra empty line is there in output.
		require.Equal(t, 3, len(lines), "expected header and warning message of the output to remain")
	}
}

func httpEndpointsTestOnUninstall(opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		// If --all, then the below does not need to run.
		if opts.UninstallAll {
			// Note: Namespace is deleted in the uninstall components function,
			// so this should return as there is nothing to delete or do.
			return
		}
		if opts.ApplyHTTPEndpointChanges {
			// On Dapr uninstall CRDs are not removed, consequently the http endpoints will not be removed.
			output, err := spawn.Command("kubectl", "get", "httpendpoints")
			require.NoError(t, err, "expected no error on calling dapr httpendpoints")
			assert.Contains(t, output, "No resources found")

			// Manually remove httpendpoints and verify output.
			output, err = spawn.Command("kubectl", "delete", "-f", "../testdata/httpendpoint.yaml")
			require.NoError(t, err, "expected no error on kubectl delete")
			require.Equal(t, "httpendpoints.dapr.io \"httpendpint\" deleted\nhttpendpoints.dapr.io \"httpendpoint\" deleted\n", output, "expected output to match")
			output, err = spawn.Command("kubectl", "delete", "-f", "../testdata/namespace.yaml")
			require.NoError(t, err, "expected no error on kubectl delete")
			t.Log(output)
			output, err = spawn.Command("kubectl", "get", "httpendpoints")
			require.NoError(t, err, "expected no error on calling dapr httpendpoints")
			lines := strings.Split(output, "\n")

			// An extra empty line is there in output.
			require.Equal(t, 2, len(lines), "expected kubernetes response message to remain")
		}
	}
}

func statusTestOnUninstall() func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := GetDaprPath()
		output, err := spawn.Command(daprPath, "status", "-k")
		t.Log("checking status fails as expected")
		require.Error(t, err, "status check did not fail as expected")
		require.Contains(t, output, " No status returned. Is Dapr initialized in your cluster?", "error on message verification")
	}
}

func componentOutputCheck(t *testing.T, output string, all bool) {
	output = strings.TrimSpace(output) // remove empty string.
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		t.Logf("num:%d line:%+v", i, line)
	}

	if all {
		assert.Equal(t, 2, len(lines), "expected at 0 components and 2 output lines")
		return
	}

	lines = strings.Split(output, "\n")[2:] // remove header and warning message.

	assert.Equal(t, 2, len(lines), "expected 2 components") // default and test namespace components.

	// for fresh cluster only one component yaml has been applied.
	testNsFields := strings.Fields(lines[0])
	defaultNsFields := strings.Fields(lines[1])

	// Fields splits on space, so Created time field might be split again.
	namespaceComponentOutputCheck(t, testNsFields, "test")
	namespaceComponentOutputCheck(t, defaultNsFields, "default")
}

func namespaceComponentOutputCheck(t *testing.T, fields []string, namespace string) {
	assert.GreaterOrEqual(t, len(fields), 6, "expected at least 6 fields in components output")
	assert.Equal(t, namespace, fields[0], "expected name to match")
	assert.Equal(t, "statestore", fields[1], "expected name to match")
	assert.Equal(t, "state.redis", fields[2], "expected type to match")
	assert.Equal(t, "v1", fields[3], "expected version to match")
	assert.Equal(t, "app1", fields[4], "expected scopes to match")
}

func httpEndpointOutputCheck(t *testing.T, output string) {
	const (
		headerName = "NAME"
		headerAge  = "AGE"
	)
	assert.Contains(t, output, headerName)
	assert.Contains(t, output, headerAge)
	// check for test httpendpoint named httpendpoint output to be present in output.
	assert.Contains(t, output, "httpendpoint")
}

func validatePodsOnInstallUpgrade(t *testing.T, details VersionDetails) {
	ctx := context.Background()
	ctxt, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	k8sClient, err := getClient()
	require.NoError(t, err)
	list, err := k8sClient.CoreV1().Pods(DaprTestNamespace).List(ctxt, v1.ListOptions{
		Limit: 100,
	})
	require.NoError(t, err)

	notFound := map[string]string{
		"sentry":  details.RuntimeVersion,
		"sidecar": details.RuntimeVersion,
		// "dashboard": details.DashboardVersion, TODO: enable when helm charts are updated.
		"placement": details.RuntimeVersion,
		"operator":  details.RuntimeVersion,
	}

	if details.ImageVariant != "" {
		notFound["sentry"] = notFound["sentry"] + "-" + details.ImageVariant
		notFound["sidecar"] = notFound["sidecar"] + "-" + details.ImageVariant
		notFound["placement"] = notFound["placement"] + "-" + details.ImageVariant
		notFound["operator"] = notFound["operator"] + "-" + details.ImageVariant
	}

	prefixes := map[string]string{
		"sentry":  "dapr-sentry-",
		"sidecar": "dapr-sidecar-injector-",
		// "dashboard": "dapr-dashboard-", TODO: enable when helm charts are updated.
		"placement": "dapr-placement-server-",
		"operator":  "dapr-operator-",
	}

	t.Logf("items %d", len(list.Items))
	for _, pod := range list.Items {
		t.Log(pod.ObjectMeta.Name)
		for component, prefix := range prefixes {
			if pod.Status.Phase != core_v1.PodRunning {
				continue
			}
			if !pod.Status.ContainerStatuses[0].Ready {
				continue
			}
			if strings.HasPrefix(pod.ObjectMeta.Name, prefix) {
				expectedVersion, ok := notFound[component]
				if !ok {
					continue
				}
				if len(pod.Spec.Containers) == 0 {
					continue
				}

				image := pod.Spec.Containers[0].Image
				versionIndex := strings.LastIndex(image, ":")
				if versionIndex != -1 {
					version := image[versionIndex+1:]
					if version == expectedVersion {
						delete(notFound, component)
					}
				}
			}
		}
	}
	assert.Empty(t, notFound)
}

func waitPodDeletion(t *testing.T, done, podsDeleted chan struct{}) {
	for {
		select {
		case <-done: // if timeout was reached.
			return
		default:
			break
		}
		ctx := context.Background()
		ctxt, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		k8sClient, err := getClient()
		require.NoError(t, err, "error getting k8s client for pods check")
		list, err := k8sClient.CoreV1().Pods(DaprTestNamespace).List(ctxt, v1.ListOptions{
			Limit: 100,
		})
		require.NoError(t, err, "error getting pods list from k8s")
		if len(list.Items) == 0 {
			podsDeleted <- struct{}{}
		}
		time.Sleep(15 * time.Second)
	}
}

func waitAllPodsRunning(t *testing.T, namespace string, haEnabled bool, done, podsRunning chan struct{}) {
	for {
		select {
		case <-done: // if timeout was reached.
			return
		default:
			break
		}
		ctx := context.Background()
		ctxt, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		k8sClient, err := getClient()
		require.NoError(t, err, "error getting k8s client for pods check")
		list, err := k8sClient.CoreV1().Pods(namespace).List(ctxt, v1.ListOptions{
			Limit: 100,
		})
		require.NoError(t, err, "error getting pods list from k8s")
		countOfReadyPods := 0
		for _, item := range list.Items {
			// Check pods running, and containers ready.
			if item.Status.Phase == core_v1.PodRunning && len(item.Status.ContainerStatuses) != 0 {
				size := len(item.Status.ContainerStatuses)
				for _, status := range item.Status.ContainerStatuses {
					if status.Ready {
						size--
					}
				}
				if size == 0 {
					countOfReadyPods++
				}
			}
		}
		if len(list.Items) == countOfReadyPods && ((haEnabled && countOfReadyPods == numHAPods) || (!haEnabled && countOfReadyPods == numNonHAPods)) {
			podsRunning <- struct{}{}
		}

		time.Sleep(15 * time.Second)
	}
}

func exportCurrentCertificate(daprPath string) error {
	_, err := os.Stat("./certs")
	if err != nil {
		os.RemoveAll("./certs")
	}
	_, err = spawn.Command(daprPath, "mtls", "export", "-o", "./certs")

	if err != nil {
		return fmt.Errorf("error in exporting certificate %w", err)
	}
	_, err = os.Stat("./certs/ca.crt")
	if err != nil {
		return fmt.Errorf("error in exporting certificate %w", err)
	}
	_, err = os.Stat("./certs/issuer.crt")
	if err != nil {
		return fmt.Errorf("error in exporting certificate %w", err)
	}
	_, err = os.Stat("./certs/issuer.key")
	if err != nil {
		return fmt.Errorf("error in exporting certificate %w", err)
	}
	return nil
}
