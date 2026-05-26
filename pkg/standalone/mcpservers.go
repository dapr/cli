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
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/util/yaml"

	mcpserverapi "github.com/dapr/dapr/pkg/apis/mcpserver/v1alpha1"
)

// ListMCPServers walks the given resources directory and returns every
// MCPServer YAML resource it finds. Empty resourcesPath defaults to
// $HOME/.dapr/components. A missing directory returns an empty list
// (matches the Kubernetes lister when the CRD isn't installed).
//
// TODO(@sicoyle): replace with `loader.NewMCPServerLocalLoader(...).Load(ctx)`
// once dapr/dapr exposes its `pkg/internal/loader/disk/mcpservers.go` publicly
// (the way `pkg/components/loader/localloader.go` already does for Components).
func ListMCPServers(resourcesPath string) (*mcpserverapi.MCPServerList, error) {
	if resourcesPath == "" {
		daprPath, err := GetDaprRuntimePath("")
		if err != nil {
			return nil, fmt.Errorf("resolve dapr runtime path: %w", err)
		}
		resourcesPath = GetDaprComponentsPath(daprPath)
	}

	out := &mcpserverapi.MCPServerList{Items: []mcpserverapi.MCPServer{}}

	info, err := os.Stat(resourcesPath)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", resourcesPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("resources path %q is not a directory", resourcesPath)
	}

	walkErr := filepath.WalkDir(resourcesPath, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %q: %w", p, err)
		}

		dec := yaml.NewYAMLToJSONDecoder(bytes.NewReader(data))
		for {
			var server mcpserverapi.MCPServer
			if err := dec.Decode(&server); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return fmt.Errorf("parse %q: %w", p, err)
			}
			// TypeMeta.Kind reads the YAML's `kind:`; `.Kind` is a method that
			// shadows it and always returns the package const.
			if server.TypeMeta.Kind != mcpserverapi.Kind {
				continue
			}
			out.Items = append(out.Items, server)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	return out, nil
}
