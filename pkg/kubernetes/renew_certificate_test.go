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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func sentryPod(name, namespace string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": "dapr-sentry"},
		},
	}
}

func TestResolveControlPlaneNamespaceExplicit(t *testing.T) {
	// An explicitly provided namespace must be returned verbatim, without
	// touching the cluster.
	ns, err := resolveControlPlaneNamespace("my-dapr-system")
	require.NoError(t, err)
	assert.Equal(t, "my-dapr-system", ns)
}

func TestSelectControlPlaneNamespace(t *testing.T) {
	t.Run("none installed", func(t *testing.T) {
		_, err := selectControlPlaneNamespace(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dapr is not installed")
	})

	t.Run("single control plane", func(t *testing.T) {
		ns, err := selectControlPlaneNamespace([]string{"dapr-system"})
		require.NoError(t, err)
		assert.Equal(t, "dapr-system", ns)
	})

	t.Run("multiple control planes is ambiguous", func(t *testing.T) {
		_, err := selectControlPlaneNamespace([]string{"team-a", "team-b"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--namespace")
	})
}

func TestListControlPlaneNamespaces(t *testing.T) {
	objects := []runtime.Object{
		sentryPod("dapr-sentry-1", "team-a"),
		sentryPod("dapr-sentry-2", "team-a"), // replica in same namespace, deduped
		sentryPod("dapr-sentry-3", "team-b"),
		// A non-sentry pod that must be ignored.
		&v1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name: "dapr-operator", Namespace: "team-c",
			Labels: map[string]string{"app": "dapr-operator"},
		}},
	}
	client := fake.NewSimpleClientset(objects...)

	namespaces, err := listControlPlaneNamespaces(client)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"team-a", "team-b"}, namespaces)
}
