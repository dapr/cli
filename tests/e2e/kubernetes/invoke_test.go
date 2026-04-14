//go:build e2e
// +build e2e

/*
Copyright 2024 The Dapr Authors
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
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/dapr/cli/tests/e2e/common"
	"github.com/dapr/cli/tests/e2e/spawn"
)

const (
	// invokeAppID is the Dapr app id used by the test app.
	invokeAppID = "invoke-e2e-app"
	// invokeAppNamespace keeps the test app isolated from the Dapr control plane namespace.
	invokeAppNamespace = "dapr-cli-invoke-test"
	// invokeAppImage is a minimal HTTP server that returns a known body on GET /.
	// nginx is used because it is pulled by KinD base images and does not require Dapr SDK wiring:
	// the cli's `invoke -k` path targets the pod's appPort directly, not daprd, so any HTTP server works.
	invokeAppImage = "nginx:1.27-alpine"
	// invokeAppPort is the container port nginx listens on and the value we annotate as dapr.io/app-port.
	invokeAppPort = 80
)

// TestKubernetesInvoke verifies `dapr invoke -k` against a daprized app running in a KinD cluster.
//
// Flow:
//  1. Ensure a clean environment and install Dapr in the default test namespace.
//  2. Deploy a minimal HTTP server pod annotated for Dapr into a dedicated namespace.
//  3. Wait for the pod + daprd sidecar to be Ready so `dapr list -k` can discover it.
//  4. Run `dapr invoke -k --app-id ... --method ... --verb GET` and assert the response
//     contains the expected payload served by the app. This exercises the codepath in
//     pkg/kubernetes/invoke.go which proxies through the API server to appPort.
//  5. Tear the app and Dapr down regardless of outcome.
func TestKubernetesInvoke(t *testing.T) {
	if common.ShouldSkipTest(common.DaprModeNonHA) {
		t.Skipf("Skipping %s mode test", common.DaprModeNonHA)
	}

	ensureCleanEnv(t, false)

	// Install Dapr first – invoke -k requires the operator / sidecar injector to be present.
	installTests := common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           false,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})
	for _, tc := range installTests {
		t.Run(tc.Name, tc.Callable)
		if t.Failed() {
			return
		}
	}

	// Make sure the test app and Dapr are always removed, even if the body fails mid-way.
	t.Cleanup(func() {
		// best-effort: log but do not fail the test on teardown errors.
		if err := deleteInvokeTestApp(); err != nil {
			t.Logf("failed to delete test app: %v", err)
		}
		uninstallTests := common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
			CheckResourceExists: map[common.Resource]bool{
				common.CustomResourceDefs:  true,
				common.ClusterRoles:        false,
				common.ClusterRoleBindings: false,
			},
		})
		for _, tc := range uninstallTests {
			t.Run(tc.Name, tc.Callable)
		}
	})

	t.Run("deploy test app", func(t *testing.T) {
		require.NoError(t, deployInvokeTestApp(t))
	})

	t.Run("invoke via kubernetes flag", func(t *testing.T) {
		daprPath := common.GetDaprPath()
		// Retry invoke because the daprd sidecar may take a few seconds after the app
		// pod reports Ready before the dapr operator/list path can see it.
		var (
			output string
			err    error
		)
		deadline := time.Now().Add(90 * time.Second)
		for time.Now().Before(deadline) {
			output, err = spawn.Command(daprPath,
				"invoke",
				"--log-as-json",
				"-k",
				"--app-id", invokeAppID,
				"--method", "/",
				"--verb", "GET",
			)
			if err == nil && strings.Contains(output, "nginx") {
				break
			}
			time.Sleep(3 * time.Second)
		}
		require.NoError(t, err, "dapr invoke -k failed. output:\n%s", output)
		// nginx welcome page contains the string "Welcome to nginx!" – match loosely so
		// a future version bump does not break the assertion.
		assert.Contains(t, output, "nginx", "unexpected invoke output:\n%s", output)
	})

	t.Run("invoke unknown app returns error", func(t *testing.T) {
		daprPath := common.GetDaprPath()
		output, err := spawn.Command(daprPath,
			"invoke",
			"--log-as-json",
			"-k",
			"--app-id", "this-app-does-not-exist",
			"--method", "/",
			"--verb", "GET",
		)
		require.Error(t, err, "invoke against a missing app should fail. output:\n%s", output)
		assert.Contains(t, output, "not found")
	})
}

// deployInvokeTestApp creates a dedicated namespace and a daprized nginx deployment whose pod
// is annotated with the appID + appPort the invoke test expects. It blocks until the pod is Ready.
func deployInvokeTestApp(t *testing.T) error {
	client, err := newKubeClient()
	if err != nil {
		return fmt.Errorf("build kube client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// 1. Namespace – create if missing, reuse otherwise so repeated runs on a dirty cluster still work.
	ns := &core_v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: invokeAppNamespace},
	}
	if _, err := client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("create namespace: %w", err)
		}
	}

	// 2. Deployment – a single-replica nginx with Dapr annotations so the injector adds daprd.
	replicas := int32(1)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      invokeAppID,
			Namespace: invokeAppNamespace,
			Labels:    map[string]string{"app": invokeAppID},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": invokeAppID}},
			Template: core_v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": invokeAppID},
					Annotations: map[string]string{
						"dapr.io/enabled":  "true",
						"dapr.io/app-id":   invokeAppID,
						"dapr.io/app-port": fmt.Sprintf("%d", invokeAppPort),
					},
				},
				Spec: core_v1.PodSpec{
					Containers: []core_v1.Container{{
						Name:  "app",
						Image: invokeAppImage,
						Ports: []core_v1.ContainerPort{{
							ContainerPort: invokeAppPort,
							Protocol:      core_v1.ProtocolTCP,
						}},
						ReadinessProbe: &core_v1.Probe{
							ProbeHandler: core_v1.ProbeHandler{
								TCPSocket: &core_v1.TCPSocketAction{
									Port: intstr.FromInt(invokeAppPort),
								},
							},
							InitialDelaySeconds: 1,
							PeriodSeconds:       2,
						},
					}},
				},
			},
		},
	}
	if _, err := client.AppsV1().Deployments(invokeAppNamespace).Create(ctx, deploy, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("create deployment: %w", err)
		}
	}

	// 3. Wait for at least one pod to be Ready and carry the daprd sidecar so `dapr invoke -k` can find it.
	return waitForInvokeAppReady(ctx, t, client)
}

func waitForInvokeAppReady(ctx context.Context, t *testing.T, client k8s.Interface) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		pods, err := client.CoreV1().Pods(invokeAppNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=" + invokeAppID,
		})
		if err == nil {
			for _, pod := range pods.Items {
				if pod.Status.Phase != core_v1.PodRunning {
					continue
				}
				if !allContainersReady(pod) {
					continue
				}
				if !hasDaprdSidecar(pod) {
					continue
				}
				t.Logf("test app pod ready: %s (%d containers)", pod.Name, len(pod.Status.ContainerStatuses))
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for test app pod to become ready: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func allContainersReady(pod core_v1.Pod) bool {
	if len(pod.Status.ContainerStatuses) == 0 {
		return false
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			return false
		}
	}
	return true
}

func hasDaprdSidecar(pod core_v1.Pod) bool {
	for _, c := range pod.Spec.Containers {
		if c.Name == "daprd" {
			return true
		}
	}
	return false
}

// deleteInvokeTestApp tears down the namespace created for the test app. Deletion is synchronous
// enough for the subsequent Dapr uninstall to succeed; we do not wait for it to complete.
func deleteInvokeTestApp() error {
	client, err := newKubeClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := client.CoreV1().Namespaces().Delete(ctx, invokeAppNamespace, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

// newKubeClient builds a Kubernetes client from the test environment's kubeconfig. It mirrors
// the behaviour of common.getClient() but is kept local because that helper is unexported.
func newKubeClient() (*k8s.Clientset, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, &clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, err
	}
	return k8s.NewForConfig(config)
}
