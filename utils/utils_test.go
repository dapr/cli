/*
Copyright 2022 The Dapr Authors
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

package utils

import "testing"

func TestContainerRuntimeUtils(t *testing.T) {
	testcases := []struct {
		name     string
		input    string
		expected string
		valid    bool
	}{
		{
			name:     "podman runtime is valid, and is returned as is",
			input:    "podman",
			expected: "podman",
			valid:    true,
		},
		{
			name:     "podman runtime with extra spaces is valid, and is returned as is",
			input:    "  podman  ",
			expected: "podman",
			valid:    true,
		},
		{
			name:     "docker runtime is valid, and is returned as is",
			input:    "docker",
			expected: "docker",
			valid:    true,
		},
		{
			name:     "docker runtime with extra spaces is valid, and is returned as is",
			input:    "     docker  ",
			expected: "docker",
			valid:    true,
		},
		{
			name:     "empty runtime is invalid, and docker is returned as default",
			input:    "",
			expected: "docker",
			valid:    false,
		},
		{
			name:     "invalid runtime is invalid, and docker is returned as default",
			input:    "invalid",
			expected: "docker",
			valid:    false,
		},
		{
			name:     "invalid runtime with extra spaces is invalid, and docker is returned as default",
			input:    "   invalid  ",
			expected: "docker",
			valid:    false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualValid := IsValidContainerRuntime(tc.input)
			if actualValid != tc.valid {
				t.Errorf("expected %v, got %v", tc.valid, actualValid)
			}

			actualCmd := GetContainerRuntimeCmd(tc.input)
			if actualCmd != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, actualCmd)
			}
		})
	}
}
