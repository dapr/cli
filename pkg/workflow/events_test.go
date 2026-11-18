/*
Copyright 2026 The Dapr Authors
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

package workflow

import (
	"testing"

	"github.com/dapr/durabletask-go/api/protos"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminateWorkflowOptionsUseExistingInput(t *testing.T) {
	tests := []struct {
		name   string
		output *string
		reason *string
		want   string
	}{
		{name: "output", output: terminateStringPtr("output"), want: "output"},
		{name: "reason", reason: stringPtr("operator requested termination"), want: "operator requested termination"},
		{name: "explicit empty reason", reason: stringPtr(""), want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := terminateWorkflowOptions(tc.output, tc.reason)
			require.NoError(t, err)
			require.Len(t, opts, 1)

			req := new(protos.TerminateRequest)
			require.NoError(t, opts[0](req))
			assert.Equal(t, tc.want, req.GetOutput().GetValue())
		})
	}
}

func TestTerminateWorkflowOptionsRejectSeparateOutputAndReason(t *testing.T) {
	_, err := terminateWorkflowOptions(stringPtr("output"), stringPtr("reason"))
	require.ErrorContains(t, err, "--output and --reason cannot be used together")
}

func stringPtr(value string) *string {
	return &value
}

