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

package standalone

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const weatherYAML = `apiVersion: dapr.io/v1alpha1
kind: MCPServer
metadata:
  name: weather
spec:
  endpoint:
    streamableHTTP:
      url: http://localhost:8081/mcp
`

const localToolsYAML = `apiVersion: dapr.io/v1alpha1
kind: MCPServer
metadata:
  name: local-tools
spec:
  endpoint:
    stdio:
      command: python
      args: ["mcp-servers/local_tools_server.py"]
`

const notMCPYAML = `apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
spec:
  type: state.redis
  version: v1
`

const multiDocYAML = `apiVersion: dapr.io/v1alpha1
kind: MCPServer
metadata:
  name: a
spec:
  endpoint:
    streamableHTTP:
      url: http://a.example.com/mcp
---
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: filler
spec:
  type: state.redis
  version: v1
---
apiVersion: dapr.io/v1alpha1
kind: MCPServer
metadata:
  name: b
spec:
  endpoint:
    sse:
      url: http://b.example.com/sse
`

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
}

func TestListMCPServers_NonexistentPath(t *testing.T) {
	list, err := ListMCPServers(filepath.Join(t.TempDir(), "does-not-exist"))
	require.NoError(t, err)
	require.NotNil(t, list)
	assert.Empty(t, list.Items, "missing directory should yield empty list, not error")
}

func TestListMCPServers_EmptyDir(t *testing.T) {
	list, err := ListMCPServers(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, list.Items)
}

func TestListMCPServers_PathIsAFile(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "config.yaml")
	require.NoError(t, os.WriteFile(file, []byte(weatherYAML), 0o600))

	_, err := ListMCPServers(file)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestListMCPServers_SkipsNonMCPYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "statestore.yaml", notMCPYAML)
	writeFile(t, dir, "weather.yaml", weatherYAML)

	list, err := ListMCPServers(dir)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "weather", list.Items[0].GetName())
	assert.NotNil(t, list.Items[0].Spec.Endpoint.StreamableHTTP)
	assert.Equal(t, "http://localhost:8081/mcp", list.Items[0].Spec.Endpoint.StreamableHTTP.URL)
}

func TestListMCPServers_MultiServerDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "weather.yaml", weatherYAML)
	writeFile(t, dir, "local-tools.yaml", localToolsYAML)
	writeFile(t, dir, "statestore.yaml", notMCPYAML)

	list, err := ListMCPServers(dir)
	require.NoError(t, err)
	require.Len(t, list.Items, 2)

	names := []string{list.Items[0].GetName(), list.Items[1].GetName()}
	assert.ElementsMatch(t, []string{"weather", "local-tools"}, names)
}

func TestListMCPServers_MultiDocFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "all.yaml", multiDocYAML)

	list, err := ListMCPServers(dir)
	require.NoError(t, err)
	require.Len(t, list.Items, 2, "two MCPServer docs + one Component should yield 2 entries")

	got := map[string]bool{}
	for _, s := range list.Items {
		got[s.GetName()] = true
	}
	assert.True(t, got["a"], "missing MCPServer a")
	assert.True(t, got["b"], "missing MCPServer b")
}

func TestListMCPServers_HonorsYmlExtension(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "weather.yml", weatherYAML)

	list, err := ListMCPServers(dir)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "weather", list.Items[0].GetName())
}

func TestListMCPServers_IgnoresNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "README.md", "# not yaml\n")
	writeFile(t, dir, "weather.yaml", weatherYAML)

	list, err := ListMCPServers(dir)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
}

func TestListMCPServers_WalksSubdirectories(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "nested")
	require.NoError(t, os.MkdirAll(sub, 0o700))
	writeFile(t, sub, "weather.yaml", weatherYAML)

	list, err := ListMCPServers(dir)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "weather", list.Items[0].GetName())
}
