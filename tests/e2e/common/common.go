// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

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

	"github.com/dapr/cli/tests/e2e/spawn"

	k8s "k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Resource int

const (
	CustomResourceDefs Resource = iota
	ClusterRoles
	ClusterRoleBindings
)

const DaprTestNamespace = "dapr-cli-tests"

type VersionDetails struct {
	RuntimeVersion      string
	DashboardVersion    string
	CustomResourceDefs  []string
	ClusterRoles        []string
	ClusterRoleBindings []string
}
type TestOptions struct {
	HAEnabled             bool
	MTLSEnabled           bool
	ApplyComponentChanges bool
	CheckResourceExists   map[Resource]bool
	UninstallAll          bool
}

type TestCase struct {
	Name     string
	Callable func(*testing.T)
}

func UpgradeTest(details VersionDetails) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := getDaprPath()
		args := []string{
			"upgrade", "-k",
			"--runtime-version", details.RuntimeVersion,
			"--log-as-json"}
		output, err := spawn.Command(daprPath, args...)
		t.Log(output)
		require.NoError(t, err, "upgrade failed")

		done := make(chan struct{})
		podsRunning := make(chan struct{})

		go waitAllPodsRunning(t, DaprTestNamespace, done, podsRunning)
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
	daprPath := getDaprPath()

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
				// CRD already deleted and not found
				require.Contains(t, output, "Error from server (NotFound)")
				continue
			} else {
				require.NoErrorf(t, err, "expected no error on deleting crd %s", crd)
			}
			require.Equal(t, fmt.Sprintf("customresourcedefinition.apiextensions.k8s.io \"%s\" deleted\n", crd), output, "expected output to match")
		}
	}
}

// Get Test Cases

func GetTestsOnInstall(details VersionDetails, opts TestOptions) []TestCase {
	return []TestCase{
		{"install " + details.RuntimeVersion, installTest(details, opts)},
		{"crds exist " + details.RuntimeVersion, CRDTest(details, opts)},
		{"clusterroles exist " + details.RuntimeVersion, ClusterRolesTest(details, opts)},
		{"clusterrolebindings exist " + details.RuntimeVersion, ClusterRoleBindingsTest(details, opts)},
		{"apply and check components exist " + details.RuntimeVersion, ComponentsTestOnInstallUpgrade(opts)},
		{"check mtls " + details.RuntimeVersion, MTLSTestOnInstallUpgrade(opts)},
		{"status check " + details.RuntimeVersion, StatusTestOnInstallUpgrade(details, opts)},
	}
}

func GetTestsOnUninstall(details VersionDetails, opts TestOptions) []TestCase {
	return []TestCase{
		{"uninstall " + details.RuntimeVersion, uninstallTest(opts.UninstallAll)}, // waits for pod deletion
		{"crds exist on uninstall " + details.RuntimeVersion, CRDTest(details, opts)},
		{"clusterroles not exist " + details.RuntimeVersion, ClusterRolesTest(details, opts)},
		{"clusterrolebindings not exist " + details.RuntimeVersion, ClusterRoleBindingsTest(details, opts)},
		{"check components exist on uninstall " + details.RuntimeVersion, componentsTestOnUninstall(opts.UninstallAll)},
		{"check mtls error " + details.RuntimeVersion, uninstallMTLSTest()},
		{"check status error " + details.RuntimeVersion, statusTestOnUninstall()},
	}
}

func MTLSTestOnInstallUpgrade(opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := getDaprPath()
		output, err := spawn.Command(daprPath, "mtls", "-k")
		require.NoError(t, err, "expected no error on querying for mtls")
		if !opts.MTLSEnabled {
			t.Log("check mtls disabled")
			require.Contains(t, output, "Mutual TLS is disabled in your Kubernetes cluster", "expected output to match")
		} else {
			t.Log("check mtls enabled")
			require.Contains(t, output, "Mutual TLS is enabled in your Kubernetes cluster", "expected output to match")
		}

		// expiry
		output, err = spawn.Command(daprPath, "mtls", "expiry")
		require.NoError(t, err, "expected no error on querying for mtls expiry")
		assert.Contains(t, output, "Root certificate expires in", "expected output to contain string")
		assert.Contains(t, output, "Expiry date:", "expected output to contain string")

		// export
		// check that the dir does not exist now
		_, err = os.Stat("./certs")
		if assert.Error(t, err) {
			assert.True(t, os.IsNotExist(err), err.Error())
		}

		output, err = spawn.Command(daprPath, "mtls", "export", "-o", "./certs")
		require.NoError(t, err, "expected no error on mtls export")
		require.Contains(t, output, "Trust certs successfully exported to", "expected output to contain string")

		// check export success
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
		daprPath := getDaprPath()
		// if dapr is installed
		if opts.ApplyComponentChanges {
			// apply any changes to the component
			t.Log("apply component changes")
			output, err := spawn.Command("kubectl", "apply", "-f", "../testdata/statestore.yaml")
			require.NoError(t, err, "expected no error on kubectl apply")
			require.Equal(t, "component.dapr.io/statestore created\n", output, "expceted output to match")
		}

		t.Log("check applied component exists")
		output, err := spawn.Command(daprPath, "components", "-k")
		require.NoError(t, err, "expected no error on calling dapr components")
		componentOutputCheck(t, output, false)
	}
}

func StatusTestOnInstallUpgrade(details VersionDetails, opts TestOptions) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := getDaprPath()
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

		lines := strings.Split(output, "\n")[1:] // remove header of status
		t.Logf("dapr status -k infos: \n%s\n", lines)
		for _, line := range lines {
			cols := strings.Fields(strings.TrimSpace(line))
			if len(cols) > 6 { // atleast 6 fields are verified from status (Age and created time are not)
				if toVerify, ok := notFound[cols[0]]; ok { // get by name
					require.Equal(t, DaprTestNamespace, cols[1], "namespace must match")
					require.Equal(t, "True", cols[2], "healthly field must be true")
					require.Equal(t, "Running", cols[3], "pods must be Running")
					require.Equal(t, toVerify[1], cols[4], "replicas must be equal")
					require.Equal(t, toVerify[0], cols[5], "versions must match")
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

// Unexported functions

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
func getDaprPath() string {
	distDir := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	return filepath.Join("..", "..", "..", "dist", distDir, "release", "dapr")
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
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
		daprPath := getDaprPath()
		args := []string{
			"init", "-k",
			"--wait",
			"-n", DaprTestNamespace,
			"--runtime-version", details.RuntimeVersion,
			"--log-as-json"}
		if opts.HAEnabled {
			args = append(args, "--enable-ha")
		}
		if !opts.MTLSEnabled {
			t.Log("install without mtls")
			args = append(args, "--enable-mtls=false")
		} else {
			t.Log("install with mtls")
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
		// wait for pods to be deleted completely
		// needed to verify status checks fails correctly
		podsDeleted := make(chan struct{})
		done := make(chan struct{})
		t.Log("waiting for pods to be deleted completely")
		go waitPodDeletion(t, done, podsDeleted)
		select {
		case <-podsDeleted:
			t.Log("pods were delted as expected on uninstall")
			return
		case <-time.After(2 * time.Minute):
			done <- struct{}{}
			t.Error("timeout verifying pods were deleted as expectedx")
		}
	}
}

func uninstallMTLSTest() func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := getDaprPath()
		output, err := spawn.Command(daprPath, "mtls", "-k")
		require.Error(t, err, "expected error to be return if dapr not installed")
		require.Contains(t, output, "error checking mTLS: system configuration not found", "expected output to match")
	}
}

func componentsTestOnUninstall(all bool) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := getDaprPath()
		// On Dapr uninstall CRDs are not removed, consequently the components will not be removed
		// TODO Related to https://github.com/dapr/cli/issues/656
		// For now the components remain
		output, err := spawn.Command(daprPath, "components", "-k")
		require.NoError(t, err, "expected no error on calling dapr components")
		componentOutputCheck(t, output, all)

		// If --all, then the below does not need to run.
		if all {
			return
		}

		// Manually remove components and verify output
		output, err = spawn.Command("kubectl", "delete", "-f", "../testdata/statestore.yaml")
		require.NoError(t, err, "expected no error on kubectl apply")
		require.Equal(t, "component.dapr.io \"statestore\" deleted\n", output, "expected output to match")
		output, err = spawn.Command(daprPath, "components", "-k")
		require.NoError(t, err, "expected no error on calling dapr components")
		lines := strings.Split(output, "\n")

		// An extra empty line is there in output
		require.Equal(t, 2, len(lines), "expected only header of the output to remain")
	}
}

func statusTestOnUninstall() func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := getDaprPath()
		output, err := spawn.Command(daprPath, "status", "-k")
		t.Log("checking status fails as expected")
		require.Error(t, err, "status check did not fail as expected")
		require.Contains(t, output, " No status returned. Is Dapr initialized in your cluster?", "error on message verification")
	}
}

func componentOutputCheck(t *testing.T, output string, all bool) {
	lines := strings.Split(output, "\n")[1:] // remove header
	// for fresh cluster only one component yaml has been applied
	fields := strings.Fields(lines[0])

	if all {
		assert.Equal(t, len(fields), 0, "expected at 0 components output")

		return
	}

	// Fields splits on space, so Created time field might be split again
	assert.GreaterOrEqual(t, len(fields), 6, "expected at least 6 fields in components output")
	assert.Equal(t, "statestore", fields[0], "expected name to match")
	assert.Equal(t, "state.redis", fields[1], "expected type to match")
	assert.Equal(t, "v1", fields[2], "expected version to match")
	assert.Equal(t, "app1", fields[3], "expected scopes to match")
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
		"sentry":    details.RuntimeVersion,
		"sidecar":   details.RuntimeVersion,
		"dashboard": details.DashboardVersion,
		"placement": details.RuntimeVersion,
		"operator":  details.RuntimeVersion,
	}
	prefixes := map[string]string{
		"sentry":    "dapr-sentry-",
		"sidecar":   "dapr-sidecar-injector-",
		"dashboard": "dapr-dashboard-",
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
		case <-done: // if timeout was reached
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

func waitAllPodsRunning(t *testing.T, namespace string, done, podsRunning chan struct{}) {
	for {
		select {
		case <-done: // if timeout was reached
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
		count := 0
		for _, item := range list.Items {
			// Check pods running, and containers ready
			if item.Status.Phase == core_v1.PodRunning && len(item.Status.ContainerStatuses) != 0 && item.Status.ContainerStatuses[0].Ready {
				count++
			}
		}
		if len(list.Items) == count {
			podsRunning <- struct{}{}
		}
		time.Sleep(15 * time.Second)
	}
}
