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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListPodsInterface(t *testing.T) {
	t.Run("empty list pods", func(t *testing.T) {
		k8s := fake.NewSimpleClientset()
		output, err := ListPodsInterface(k8s, map[string]string{
			"test": "test",
		})
		assert.Nil(t, err, "unexpected error")
		assert.NotNil(t, output, "Expected empty list")
		assert.Equal(t, 0, len(output.Items), "Expected length 0")
	})
	t.Run("one matching pod", func(t *testing.T) {
		k8s := fake.NewSimpleClientset((&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test",
				Namespace:   "test",
				Annotations: map[string]string{},
				Labels: map[string]string{
					"test": "test",
				},
			},
		}))
		output, err := ListPodsInterface(k8s, map[string]string{
			"test": "test",
		})
		assert.Nil(t, err, "unexpected error")
		assert.NotNil(t, output, "Expected non empty list")
		assert.Equal(t, 1, len(output.Items), "Expected length 0")
		assert.Equal(t, "test", output.Items[0].Name, "expected name to match")
		assert.Equal(t, "test", output.Items[0].Namespace, "expected namespace to match")
	})
}
