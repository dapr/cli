// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func newTestSimpleK8s(objects ...runtime.Object) *StatusClient {
	client := StatusClient{}
	client.client = fake.NewSimpleClientset(objects...)
	return &client
}

func newDaprControlPlanePod(name string, appName string, creationTime time.Time, state v1.ContainerState, ready bool) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   "dapr-system",
			Annotations: map[string]string{},
			Labels: map[string]string{
				"app": appName,
			},
			CreationTimestamp: metav1.Time{
				Time: creationTime,
			},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					State: state,
					Ready: ready,
				},
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Image: name + ":0.0.1",
				},
			},
		},
	}
}

func TestStatus(t *testing.T) {
	t.Run("empty status. dapr not init", func(t *testing.T) {
		k8s := newTestSimpleK8s()
		status, err := k8s.Status()
		if err != nil {
			t.Fatalf("%s status should not raise an error", err.Error())
		}
		assert.Equal(t, 0, len(status), "Expected status to be empty list")
	})

	t.Run("one status waiting", func(t *testing.T) {
		k8s := newTestSimpleK8s(newDaprControlPlanePod(
			"dapr-dashboard-58877dbc9d-n8qg2", "dapr-dashboard",
			time.Now(),
			v1.ContainerState{
				Waiting: &v1.ContainerStateWaiting{
					Reason:  "test",
					Message: "test",
				},
			}, false))
		status, err := k8s.Status()
		assert.Nil(t, err, "status should not raise an error")
		assert.Equal(t, 1, len(status), "Expected status to be empty list")
		stat := status[0]
		assert.Equal(t, "dapr-dashboard", stat.Name, "expected name to match")
		assert.Equal(t, "dapr-system", stat.Namespace, "expected namespace to match")
		assert.Equal(t, "0.0.1", stat.Version, "expected version to match")
		assert.Equal(t, 1, stat.Replicas, "expected replicas to match")
		assert.Equal(t, "False", stat.Healthy, "expected health to match")
		assert.True(t, strings.HasPrefix(stat.Status, "Waiting"), "expected waiting status")
	})

	t.Run("one status running", func(t *testing.T) {
		testTime := time.Now()
		k8s := newTestSimpleK8s(newDaprControlPlanePod(
			"dapr-dashboard-58877dbc9d-n8qg2", "dapr-dashboard",
			testTime.Add(time.Duration(-20)*time.Minute),
			v1.ContainerState{
				Running: &v1.ContainerStateRunning{
					StartedAt: metav1.Time{
						Time: testTime.Add(time.Duration(-19) * time.Minute),
					},
				},
			}, true))
		status, err := k8s.Status()
		assert.Nil(t, err, "status should not raise an error")
		assert.Equal(t, 1, len(status), "Expected status to be empty list")
		stat := status[0]
		assert.Equal(t, "dapr-dashboard", stat.Name, "expected name to match")
		assert.Equal(t, "dapr-system", stat.Namespace, "expected namespace to match")
		assert.Equal(t, "20m", stat.Age, "expected age to match")
		assert.Equal(t, "0.0.1", stat.Version, "expected version to match")
		assert.Equal(t, 1, stat.Replicas, "expected replicas to match")
		assert.Equal(t, "True", stat.Healthy, "expected health to match")
		assert.Equal(t, stat.Status, "Running", "expected running status")
	})

	t.Run("one status terminated", func(t *testing.T) {
		testTime := time.Now()

		k8s := newTestSimpleK8s(newDaprControlPlanePod(
			"dapr-dashboard-58877dbc9d-n8qg2", "dapr-dashboard",
			testTime.Add(time.Duration(-20)*time.Minute),
			v1.ContainerState{
				Terminated: &v1.ContainerStateTerminated{
					ExitCode: 1,
				},
			}, false))

		status, err := k8s.Status()
		assert.Nil(t, err, "status should not raise an error")
		assert.Equal(t, 1, len(status), "Expected status to be empty list")
		stat := status[0]
		assert.Equal(t, "dapr-dashboard", stat.Name, "expected name to match")
		assert.Equal(t, "dapr-system", stat.Namespace, "expected namespace to match")
		assert.Equal(t, "20m", stat.Age, "expected age to match")
		assert.Equal(t, "0.0.1", stat.Version, "expected version to match")
		assert.Equal(t, 1, stat.Replicas, "expected replicas to match")
		assert.Equal(t, "False", stat.Healthy, "expected health to match")
		assert.Equal(t, stat.Status, "Terminated", "expected terminated status")
	})

	t.Run("one status empty client", func(t *testing.T) {
		k8s := &StatusClient{}
		status, err := k8s.Status()
		assert.NotNil(t, err, "status should raise an error")
		assert.Equal(t, "kubernetes client not initialized", err.Error(), "expected errors to match")
		assert.Nil(t, status, "expected nil for status")
	})
}

func TestControlPlaneServices(t *testing.T) {
	controlPlaneServices := []struct {
		name    string
		appName string
	}{
		{"dapr-dashboard-58877dbc9d-n8qg2", "dapr-dashboard"},
		{"dapr-operator-67d7d7bb6c-7h96c", "dapr-operator"},
		{"dapr-operator-67d7d7bb6c-2h96d", "dapr-operator"},
		{"dapr-operator-67d7d7bb6c-3h96c", "dapr-operator"},
		{"dapr-placement-server-0", "dapr-placement-server"},
		{"dapr-placement-server-1", "dapr-placement-server"},
		{"dapr-placement-server-2", "dapr-placement-server"},
		{"dapr-sentry-647759cd46-9ptks", "dapr-sentry"},
		{"dapr-sentry-647759cd46-aptks", "dapr-sentry"},
		{"dapr-sentry-647759cd46-bptks", "dapr-sentry"},
		{"dapr-sidecar-injector-74648c9dcb-5bsmn", "dapr-sidecar-injector"},
		{"dapr-sidecar-injector-74648c9dcb-6bsmn", "dapr-sidecar-injector"},
		{"dapr-sidecar-injector-74648c9dcb-7bsmn", "dapr-sidecar-injector"},
	}

	expectedReplicas := map[string]int{}

	runtimeObj := make([]runtime.Object, len(controlPlaneServices))
	for i, s := range controlPlaneServices {
		testTime := time.Now()
		runtimeObj[i] = newDaprControlPlanePod(
			s.name, s.appName,
			testTime.Add(time.Duration(-20)*time.Minute),
			v1.ContainerState{
				Running: &v1.ContainerStateRunning{
					StartedAt: metav1.Time{
						Time: testTime.Add(time.Duration(-19) * time.Minute),
					},
				},
			}, true)
		expectedReplicas[s.appName]++
	}

	k8s := newTestSimpleK8s(runtimeObj...)
	status, err := k8s.Status()
	assert.Nil(t, err, "status should not raise an error")

	assert.Equal(t, len(expectedReplicas), len(status), "Expected status to be empty list")

	for _, stat := range status {
		replicas, ok := expectedReplicas[stat.Name]
		assert.True(t, ok)
		assert.Equal(t, replicas, stat.Replicas, "expected replicas to match")
	}
}
