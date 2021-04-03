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
	uninstall()

	tests := []struct {
		name  string
		phase func(*testing.T)
	}{
		{"install", testInstall(false)},
		{"crds exist", testCRDs(true)},
		{"clusterroles exist", testClusterRoles(true)},
		{"clusterrolebindings exist", testClusterRoleBindings(true)},
		{"status check", testStatus(false)},
		//-------------------------------------------------
		{"uninstall", testUninstall},
		{"clusterroles not exist", testClusterRoles(false)},
		{"clusterroles not exist", testClusterRoles(false)},
		{"clusterrolebindings not exist", testClusterRoleBindings(false)},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.phase)
	}
}

func TestKubernetesInstallHA(t *testing.T) {
	// Ensure a clean environment
	uninstall()

	tests := []struct {
		name  string
		phase func(*testing.T)
	}{
		{"install", testInstall(true)},
		{"crds exist", testCRDs(true)},
		{"clusterroles exist", testClusterRoles(true)},
		{"clusterrolebindings exist", testClusterRoleBindings(true)},
		{"status check", testStatus(true)},
		//-------------------------------------------------
		{"uninstall", testUninstall},
		{"clusterroles not exist", testClusterRoles(false)},
		{"clusterroles not exist", testClusterRoles(false)},
		{"clusterrolebindings not exist", testClusterRoleBindings(false)},
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

func testStatus(haMode bool) func(t *testing.T) {
	return func(t *testing.T) {
		daprPath := getDaprPath()
		output, err := spawn.Command(daprPath, "status", "-k")
		require.NoError(t, err, "status check failed")
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
