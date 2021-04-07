// +build e2e

// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes_test

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

const (
	daprNamespace        = "dapr-cli-tests"
	daprRuntimeVersion   = "1.1.0"
	daprDashboardVersion = "0.6.0"
)

func TestKubernetesInstallNonHA(t *testing.T) {
	// Ensure a clean environment
	uninstall() // does not wait for pod deletion

	tests := []struct {
		name  string
		phase func(*testing.T)
	}{
		{"install without mtls", testInstall(false)},
		{"crds exist", testCRDs(true)},
		{"clusterroles exist", testClusterRoles(true)},
		{"clusterrolebindings exist", testClusterRoleBindings(true)},
		{"apply and check components exist", testComponents(true)},
		{"check mtls disabled", testMtls(true, false)},
		{"status check", testStatus(true, false)},
		//-------------------------------------------------
		{"uninstall", testUninstall}, // waits for pod deletion
		// related to https://github.com/dapr/cli/issues/656
		{"crds  exist after uninstall", testCRDs(true)},
		{"clusterroles not exist", testClusterRoles(false)},
		{"clusterrolebindings not exist", testClusterRoleBindings(false)},
		{"check components do not exist", testComponents(false)},
		{"check mtls error", testMtls(false, false)},          // second parameter does not matter here
		{"status check errors out", testStatus(false, false)}, // second parameter does not matter here
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.phase)
	}
}

func TestKubernetesInstallHA(t *testing.T) {
	// Ensure a clean environment
	uninstall() // does not wait for pod deletion

	tests := []struct {
		name  string
		phase func(*testing.T)
	}{
		{"install with mtls default", testInstall(true)},
		{"crds exist", testCRDs(true)},
		{"clusterroles exist", testClusterRoles(true)},
		{"clusterrolebindings exist", testClusterRoleBindings(true)},
		{"apply and check components exist", testComponents(true)},
		{"check mtls enabled", testMtls(true, true)},
		{"status check", testStatus(true, true)},
		//-------------------------------------------------
		{"uninstall", testUninstall}, // waits for pod deletion
		// related to https://github.com/dapr/cli/issues/656
		{"crds  exist after uninstall", testCRDs(true)},
		{"clusterroles not exist", testClusterRoles(false)},
		{"clusterrolebindings not exist", testClusterRoleBindings(false)},
		{"check components do not exist", testComponents(false)},
		{"check mtls error", testMtls(false, false)},         // second parameter does not matter here
		{"status check errors out", testStatus(false, true)}, // second parameter does not matter here
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.phase)
	}
}

func getDaprPath() string {
	distDir := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	return filepath.Join("..", "..", "..", "dist", distDir, "release", "dapr")
}

func uninstall() (string, error) {
	daprPath := getDaprPath()

	return spawn.Command(daprPath,
		"uninstall", "-k",
		"-n", daprNamespace,
		"--log-as-json")
}

func testUninstall(t *testing.T) {
	output, err := uninstall()
	t.Log(output)
	require.NoError(t, err, "uninstall failed")
	// wait for pods to be deleted completely
	// needed to verify status checks fails correctly
	podsDeleted := make(chan struct{})
	done := make(chan struct{})
	t.Log("waiting for pods to be deleted completely")
	go waitPodDeletion(done, podsDeleted, t)
	select {
	case <-podsDeleted:
		t.Log("pods were delted as expected on uninstall")
		return
	case <-time.After(2 * time.Minute):
		done <- struct{}{}
		t.Error("Pods were not deleted as expectedx")
	}
}

func testInstall(haMode bool) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := getDaprPath()
		args := []string{
			"init", "-k",
			"--wait",
			"-n", daprNamespace,
			"--runtime-version", daprRuntimeVersion,
			"--log-as-json"}
		if haMode {
			args = append(args, "--enable-ha")
		} else {
			// For now testing mtls disabled flag only for the non-HA mode
			args = append(args, "--enable-mtls=false")
		}
		output, err := spawn.Command(daprPath, args...)
		t.Log(output)
		require.NoError(t, err, "init failed")

		ctx := context.Background()
		ctxt, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		k8sClient, err := kubernetes.Client()
		require.NoError(t, err)
		list, err := k8sClient.CoreV1().Pods(daprNamespace).List(ctxt, v1.ListOptions{
			Limit: 100,
		})
		require.NoError(t, err)

		notFound := map[string]string{
			"sentry":    daprRuntimeVersion,
			"sidecar":   daprRuntimeVersion,
			"dashboard": daprDashboardVersion,
			"placement": daprRuntimeVersion,
			"operator":  daprRuntimeVersion,
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
}

func testMtls(isDaprInstalled, enabled bool) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := getDaprPath()
		output, err := spawn.Command(daprPath, "mtls", "-k")
		if !isDaprInstalled {
			require.Error(t, err, "expected error to be return if dapr not installed")
			require.Contains(t, output, "error checking mTLS: system configuration not found", "expected output to match")
			return
		}
		require.NoError(t, err, "expected no error on querying for mtls")
		if !enabled {
			require.Contains(t, output, "Mutual TLS is disabled in your Kubernetes cluster", "expected output to match")
		} else {
			require.Contains(t, output, "Mutual TLS is enabled in your Kubernetes cluster", "expected output to match")
		}

		//expiry
		output, err = spawn.Command(daprPath, "mtls", "expiry")
		require.NoError(t, err, "expected no error on querying for mtls expiry")
		assert.Contains(t, output, "Root certificate expires in", "expected output to contain string")
		assert.Contains(t, output, "Expiry date:", "expected output to contain string")

		//export
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

func testComponents(isDaprInstalled bool) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := getDaprPath()
		if isDaprInstalled {
			// if dapr is installed, apply and check for components
			output, err := spawn.Command("kubectl", "apply", "-f", "../testdata/statestore.yaml")
			require.NoError(t, err, "expected no error on kubectl apply")
			require.Equal(t, "component.dapr.io/statestore created\n", output, "expceted output to match")
			output, err = spawn.Command(daprPath, "components", "-k")
			require.NoError(t, err, "expected no error on calling dapr components")
			componentOutputCheck(t, output)
		} else {
			// On Dapr uninstall CRDs are not removed, consequently the components will not be removed
			// Related to https://github.com/dapr/cli/issues/656
			// For now the components remain
			output, err := spawn.Command(daprPath, "components", "-k")
			require.NoError(t, err, "expected no error on calling dapr components")
			componentOutputCheck(t, output)
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
}

func componentOutputCheck(t *testing.T, output string) {
	lines := strings.Split(output, "\n")[1:] // remove header
	// for fresh cluster only one component yaml has been applied
	fields := strings.Fields(lines[0])
	// Fields splits on space, so Created time field might be split again
	assert.GreaterOrEqual(t, len(fields), 6, "expected at least 6 fields in components outptu")
	assert.Equal(t, "statestore", fields[0], "expected name to match")
	assert.Equal(t, "state.redis", fields[1], "expected type to match")
	assert.Equal(t, "v1", fields[2], "expected version to match")
	assert.Equal(t, "app1", fields[3], "expected scopes to match")
}

func testStatus(isDaprInstalled, haMode bool) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := getDaprPath()
		output, err := spawn.Command(daprPath, "status", "-k")
		if isDaprInstalled {
			require.NoError(t, err, "status check failed")
		} else {
			t.Log("checking status fails as expected")
			require.Error(t, err, "status check did not fail as expected")
			require.Contains(t, output, " No status returned. Is Dapr initialized in your cluster?", "error on message verification")
			return
		}
		notFound := map[string][]string{}
		if !haMode {
			notFound = map[string][]string{
				"dapr-sentry":           {daprRuntimeVersion, "1"},
				"dapr-sidecar-injector": {daprRuntimeVersion, "1"},
				"dapr-dashboard":        {daprDashboardVersion, "1"},
				"dapr-placement-server": {daprRuntimeVersion, "1"},
				"dapr-operator":         {daprRuntimeVersion, "1"},
			}
		} else {
			notFound = map[string][]string{
				"dapr-sentry":           {daprRuntimeVersion, "3"},
				"dapr-sidecar-injector": {daprRuntimeVersion, "3"},
				"dapr-dashboard":        {daprDashboardVersion, "1"},
				"dapr-placement-server": {daprRuntimeVersion, "3"},
				"dapr-operator":         {daprRuntimeVersion, "3"},
			}
		}

		lines := strings.Split(output, "\n")[1:] // remove header of status
		for _, line := range lines {
			cols := strings.Fields(strings.TrimSpace(line))
			if len(cols) > 6 { // atleast 6 fields are verified from status (Age and created time are not)
				if toVerify, ok := notFound[cols[0]]; ok { // get by name
					require.Equal(t, daprNamespace, cols[1], "namespace must match")
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

func testCRDs(wanted bool) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		cfg, err := getConfig()
		require.NoError(t, err)

		apiextensionsClientSet, err := apiextensionsclient.NewForConfig(cfg)
		require.NoError(t, err)

		foundMap := map[string]bool{
			"components.dapr.io":     false,
			"configurations.dapr.io": false,
			"subscriptions.dapr.io":  false,
		}

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

func testClusterRoleBindings(wanted bool) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		k8sClient, err := getClient()
		require.NoError(t, err)

		foundMap := map[string]bool{
			"dapr-operator":                 false,
			"dapr-role-tokenreview-binding": false,
			"dashboard-reader-global":       false,
		}

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

func testClusterRoles(wanted bool) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		k8sClient, err := getClient()
		require.NoError(t, err)

		foundMap := map[string]bool{
			"dapr-operator-admin": false,
			"dashboard-reader":    false,
		}

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

// getClient returns a new Kubernetes client.
func getClient() (*k8s.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	return k8s.NewForConfig(config)
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

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func waitPodDeletion(done, podsDeleted chan struct{}, t *testing.T) {
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
		k8sClient, err := kubernetes.Client()
		require.NoError(t, err, "error getting k8s client for pods check")
		list, err := k8sClient.CoreV1().Pods(daprNamespace).List(ctxt, v1.ListOptions{
			Limit: 100,
		})
		require.NoError(t, err, "error getting pods list from k8s")
		if len(list.Items) == 0 {
			podsDeleted <- struct{}{}
		}
		time.Sleep(15 * time.Second)
	}
}
