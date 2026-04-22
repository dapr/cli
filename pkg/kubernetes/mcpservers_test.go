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

package kubernetes

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/dapr/dapr/pkg/apis/mcpserver/v1alpha1"
)

func TestMCPServers(t *testing.T) {
	now := meta_v1.Now()

	testCases := []struct {
		name           string
		serverName     string
		outputFormat   string
		errorExpected  bool
		errString      string
		mcpServers     []v1alpha1.MCPServer
		expectedOutput string
	}{
		{
			name:         "no MCPServers",
			outputFormat: "list",
			mcpServers:   []v1alpha1.MCPServer{},
		},
		{
			name:         "list MCPServers",
			outputFormat: "list",
			mcpServers: []v1alpha1.MCPServer{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "payments-mcp",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.MCPServerSpec{
						Endpoint: v1alpha1.MCPEndpoint{
							StreamableHTTP: &v1alpha1.MCPStreamableHTTP{
								URL: "https://payments.internal/mcp",
							},
						},
					},
				},
			},
		},
		{
			name:         "filter by name",
			serverName:   "payments-mcp",
			outputFormat: "list",
			mcpServers: []v1alpha1.MCPServer{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "payments-mcp",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.MCPServerSpec{
						Endpoint: v1alpha1.MCPEndpoint{
							StreamableHTTP: &v1alpha1.MCPStreamableHTTP{
								URL: "https://payments.internal/mcp",
							},
						},
					},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "other-mcp",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.MCPServerSpec{
						Endpoint: v1alpha1.MCPEndpoint{
							SSE: &v1alpha1.MCPSSE{
								URL: "https://other.internal/sse",
							},
						},
					},
				},
			},
		},
		{
			name:         "stdio transport",
			outputFormat: "list",
			mcpServers: []v1alpha1.MCPServer{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "local-tools",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.MCPServerSpec{
						Endpoint: v1alpha1.MCPEndpoint{
							Stdio: &v1alpha1.MCPStdio{
								Command: "npx",
								Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
							},
						},
					},
				},
			},
		},
		{
			name:          "error from API",
			outputFormat:  "list",
			errorExpected: true,
			errString:     "connection refused",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buff bytes.Buffer
			err := writeMCPServers(&buff,
				func() (*v1alpha1.MCPServerList, error) {
					if len(tc.errString) > 0 {
						return nil, assert.AnError
					}
					return &v1alpha1.MCPServerList{Items: tc.mcpServers}, nil
				}, tc.serverName, tc.outputFormat)

			if tc.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.expectedOutput != "" {
				assert.Equal(t, tc.expectedOutput, buff.String())
			}

			// For list output with servers, verify it contains the server names.
			if !tc.errorExpected && tc.outputFormat == "list" && len(tc.mcpServers) > 0 {
				output := buff.String()
				if tc.serverName != "" {
					assert.Contains(t, output, tc.serverName)
				} else {
					for _, s := range tc.mcpServers {
						assert.Contains(t, output, s.Name)
					}
				}
			}
		})
	}
}

func TestMCPTransport(t *testing.T) {
	tests := []struct {
		name   string
		server v1alpha1.MCPServer
		want   string
	}{
		{
			name: "streamable_http",
			server: v1alpha1.MCPServer{
				Spec: v1alpha1.MCPServerSpec{
					Endpoint: v1alpha1.MCPEndpoint{
						StreamableHTTP: &v1alpha1.MCPStreamableHTTP{URL: "http://example.com"},
					},
				},
			},
			want: "streamable_http",
		},
		{
			name: "sse",
			server: v1alpha1.MCPServer{
				Spec: v1alpha1.MCPServerSpec{
					Endpoint: v1alpha1.MCPEndpoint{
						SSE: &v1alpha1.MCPSSE{URL: "http://example.com"},
					},
				},
			},
			want: "sse",
		},
		{
			name: "stdio",
			server: v1alpha1.MCPServer{
				Spec: v1alpha1.MCPServerSpec{
					Endpoint: v1alpha1.MCPEndpoint{
						Stdio: &v1alpha1.MCPStdio{Command: "npx"},
					},
				},
			},
			want: "stdio",
		},
		{
			name:   "empty",
			server: v1alpha1.MCPServer{},
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mcpTransport(&tc.server)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMCPURL(t *testing.T) {
	tests := []struct {
		name   string
		server v1alpha1.MCPServer
		want   string
	}{
		{
			name: "streamable_http url",
			server: v1alpha1.MCPServer{
				Spec: v1alpha1.MCPServerSpec{
					Endpoint: v1alpha1.MCPEndpoint{
						StreamableHTTP: &v1alpha1.MCPStreamableHTTP{URL: "https://example.com/mcp"},
					},
				},
			},
			want: "https://example.com/mcp",
		},
		{
			name: "stdio command",
			server: v1alpha1.MCPServer{
				Spec: v1alpha1.MCPServerSpec{
					Endpoint: v1alpha1.MCPEndpoint{
						Stdio: &v1alpha1.MCPStdio{Command: "npx"},
					},
				},
			},
			want: "npx",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mcpURL(&tc.server)
			assert.Equal(t, tc.want, got)
		})
	}
}
