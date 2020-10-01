// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
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
		k8s := newTestSimpleK8s((&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "dapr-dashboard",
				Namespace:   "dapr-system",
				Annotations: map[string]string{},
				Labels: map[string]string{
					"app": "dapr-dashboard",
				},
				CreationTimestamp: metav1.Time{
					Time: time.Now(),
				},
			},
			Status: v1.PodStatus{
				ContainerStatuses: []v1.ContainerStatus{
					{
						State: v1.ContainerState{
							Waiting: &v1.ContainerStateWaiting{
								Reason:  "test",
								Message: "test",
							},
						},
					},
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Image: "dapr-dashboard:0.0.1",
					},
				},
			},
		}))
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
		k8s := newTestSimpleK8s((&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "dapr-dashboard",
				Namespace:   "dapr-system",
				Annotations: map[string]string{},
				Labels: map[string]string{
					"app": "dapr-dashboard",
				},
				CreationTimestamp: metav1.Time{
					Time: testTime.Add(time.Duration(-20) * time.Minute),
				},
			},
			Status: v1.PodStatus{
				ContainerStatuses: []v1.ContainerStatus{
					{
						State: v1.ContainerState{
							Running: &v1.ContainerStateRunning{
								StartedAt: metav1.Time{
									Time: testTime.Add(time.Duration(-19) * time.Minute),
								},
							},
						},
						Ready: true,
					},
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Image: "dapr-dashboard:0.0.1",
					},
				},
			},
		}))
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
		k8s := newTestSimpleK8s((&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "dapr-dashboard",
				Namespace:   "dapr-system",
				Annotations: map[string]string{},
				Labels: map[string]string{
					"app": "dapr-dashboard",
				},
				CreationTimestamp: metav1.Time{
					Time: testTime.Add(time.Duration(-20) * time.Minute),
				},
			},
			Status: v1.PodStatus{
				ContainerStatuses: []v1.ContainerStatus{
					{
						State: v1.ContainerState{
							Terminated: &v1.ContainerStateTerminated{
								ExitCode: 1,
							},
						},
					},
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Image: "dapr-dashboard:0.0.1",
					},
				},
			},
		}))
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
