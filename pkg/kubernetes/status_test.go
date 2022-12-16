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

const (
	daprImageTag          = "daprio/dapr:0.0.1"
	daprDashboardImageTag = "daprio/dashboard:0.0.1"
)

type podDetails struct {
	name      string
	appName   string
	createdAt time.Time
	state     v1.ContainerState
	ready     bool
	imageURI  string
}

func newTestSimpleK8s(objects ...runtime.Object) *StatusClient {
	client := StatusClient{}
	client.client = fake.NewSimpleClientset(objects...)
	return &client
}

func newDaprControlPlanePod(pd podDetails) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pd.name,
			Namespace:   "dapr-system",
			Annotations: map[string]string{},
			Labels: map[string]string{
				"app": pd.appName,
			},
			CreationTimestamp: metav1.Time{
				Time: pd.createdAt,
			},
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					State: pd.state,
					Ready: pd.ready,
				},
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Image: pd.imageURI,
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
		pd := podDetails{
			name:      "dapr-dashboard-58877dbc9d-n8qg2",
			appName:   "dapr-dashboard",
			createdAt: time.Now(),
			state: v1.ContainerState{
				Waiting: &v1.ContainerStateWaiting{
					Reason:  "test",
					Message: "test",
				},
			},
			ready:    false,
			imageURI: "daprio/dapr-dashboard:0.0.1",
		}
		k8s := newTestSimpleK8s(newDaprControlPlanePod(pd))
		status, err := k8s.Status()
		assert.Nil(t, err, "status should not raise an error")
		assert.Equal(t, 1, len(status), "Expected status to be non-empty list")
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
		pd := podDetails{
			name:      "dapr-dashboard-58877dbc9d-n8qg2",
			appName:   "dapr-dashboard",
			createdAt: testTime.Add(time.Duration(-20) * time.Minute),
			state: v1.ContainerState{
				Running: &v1.ContainerStateRunning{
					StartedAt: metav1.Time{
						Time: testTime.Add(time.Duration(-19) * time.Minute),
					},
				},
			},
			ready:    true,
			imageURI: "daprio/dapr-dashboard:0.0.1",
		}
		k8s := newTestSimpleK8s(newDaprControlPlanePod(pd))
		status, err := k8s.Status()
		assert.Nil(t, err, "status should not raise an error")
		assert.Equal(t, 1, len(status), "Expected status to be non-empty list")
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
		pd := podDetails{
			name:      "dapr-dashboard-58877dbc9d-n8qg2",
			appName:   "dapr-dashboard",
			createdAt: testTime.Add(time.Duration(-20) * time.Minute),
			state: v1.ContainerState{
				Terminated: &v1.ContainerStateTerminated{
					ExitCode: 1,
				},
			},
			ready:    false,
			imageURI: "daprio/dapr-dashboard:0.0.1",
		}
		k8s := newTestSimpleK8s(newDaprControlPlanePod(pd))

		status, err := k8s.Status()
		assert.Nil(t, err, "status should not raise an error")
		assert.Equal(t, 1, len(status), "Expected status to be non-empty list")
		stat := status[0]
		assert.Equal(t, "dapr-dashboard", stat.Name, "expected name to match")
		assert.Equal(t, "dapr-system", stat.Namespace, "expected namespace to match")
		assert.Equal(t, "20m", stat.Age, "expected age to match")
		assert.Equal(t, "0.0.1", stat.Version, "expected version to match")
		assert.Equal(t, 1, stat.Replicas, "expected replicas to match")
		assert.Equal(t, "False", stat.Healthy, "expected health to match")
		assert.Equal(t, stat.Status, "Terminated", "expected terminated status")
	})

	t.Run("one status pending", func(t *testing.T) {
		testTime := time.Now()
		pd := podDetails{
			name:      "dapr-dashboard-58877dbc9d-n8qg2",
			appName:   "dapr-dashboard",
			createdAt: testTime.Add(time.Duration(-20) * time.Minute),
			state: v1.ContainerState{
				Terminated: &v1.ContainerStateTerminated{
					ExitCode: 1,
				},
			},
			ready:    false,
			imageURI: "daprio/dapr-dashboard:0.0.1",
		}
		pod := newDaprControlPlanePod(pd)
		// delete pod's podstatus.
		pod.Status.ContainerStatuses = nil
		pod.Status.Phase = v1.PodPending

		k8s := newTestSimpleK8s(pod)
		status, err := k8s.Status()
		assert.Nil(t, err, "status should not raise an error")
		assert.Equal(t, 1, len(status), "Expected status to be non-empty list")
		stat := status[0]
		assert.Equal(t, "dapr-dashboard", stat.Name, "expected name to match")
		assert.Equal(t, "dapr-system", stat.Namespace, "expected namespace to match")
		assert.Equal(t, "20m", stat.Age, "expected age to match")
		assert.Equal(t, "0.0.1", stat.Version, "expected version to match")
		assert.Equal(t, 1, stat.Replicas, "expected replicas to match")
		assert.Equal(t, "False", stat.Healthy, "expected health to match")
		assert.Equal(t, stat.Status, "Pending", "expected pending status")
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
		name     string
		appName  string
		imageURI string
	}{
		{"dapr-dashboard-58877dbc9d-n8qg2", "dapr-dashboard", daprDashboardImageTag},
		{"dapr-operator-67d7d7bb6c-7h96c", "dapr-operator", daprImageTag},
		{"dapr-operator-67d7d7bb6c-2h96d", "dapr-operator", daprImageTag},
		{"dapr-operator-67d7d7bb6c-3h96c", "dapr-operator", daprImageTag},
		{"dapr-placement-server-0", "dapr-placement-server", daprImageTag},
		{"dapr-placement-server-1", "dapr-placement-server", daprImageTag},
		{"dapr-placement-server-2", "dapr-placement-server", daprImageTag},
		{"dapr-sentry-647759cd46-9ptks", "dapr-sentry", daprImageTag},
		{"dapr-sentry-647759cd46-aptks", "dapr-sentry", daprImageTag},
		{"dapr-sentry-647759cd46-bptks", "dapr-sentry", daprImageTag},
		{"dapr-sidecar-injector-74648c9dcb-5bsmn", "dapr-sidecar-injector", daprImageTag},
		{"dapr-sidecar-injector-74648c9dcb-6bsmn", "dapr-sidecar-injector", daprImageTag},
		{"dapr-sidecar-injector-74648c9dcb-7bsmn", "dapr-sidecar-injector", daprImageTag},
	}

	expectedReplicas := map[string]int{}

	runtimeObj := make([]runtime.Object, len(controlPlaneServices))
	for i, s := range controlPlaneServices {
		testTime := time.Now()
		pd := podDetails{
			name:      s.name,
			appName:   s.appName,
			createdAt: testTime.Add(time.Duration(-20) * time.Minute),
			state: v1.ContainerState{
				Running: &v1.ContainerStateRunning{
					StartedAt: metav1.Time{
						Time: testTime.Add(time.Duration(-19) * time.Minute),
					},
				},
			},
			ready:    true,
			imageURI: s.imageURI,
		}
		runtimeObj[i] = newDaprControlPlanePod(pd)
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

func TestControlPlaneVersion(t *testing.T) {
	pd := podDetails{
		name:      "dapr-sentry-647759cd46-9ptks",
		appName:   "dapr-sentry",
		createdAt: time.Now().Add(time.Duration(-20) * time.Minute),
		state: v1.ContainerState{
			Running: &v1.ContainerStateRunning{
				StartedAt: metav1.Time{
					Time: time.Now().Add(time.Duration(-19) * time.Minute),
				},
			},
		},
		ready: true,
	}
	testcases := []struct {
		imageURI        string
		expectedVersion string
	}{
		{
			imageURI:        "mockImgReg:0.0.1",
			expectedVersion: "0.0.1",
		},
		{
			imageURI:        "mockImgRegHost:mockPort:0.0.2",
			expectedVersion: "0.0.2",
		},
	}
	for _, tc := range testcases {
		pd.imageURI = tc.imageURI
		k8s := newTestSimpleK8s(newDaprControlPlanePod(pd))
		status, err := k8s.Status()
		assert.Nil(t, err, "status should not raise an error")
		assert.Equal(t, 1, len(status), "Expected status to be non-empty list")
		stat := status[0]
		assert.Equal(t, tc.expectedVersion, stat.Version, "expected version to match")
	}
}
