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
	"io"
	"os"
	"sort"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dapr/cli/pkg/age"
	"github.com/dapr/cli/utils"
	v1alpha1 "github.com/dapr/dapr/pkg/apis/mcpserver/v1alpha1"
	"github.com/dapr/dapr/pkg/client/clientset/versioned"
)

// MCPServerOutput represents an MCPServer resource for table output.
type MCPServerOutput struct {
	Namespace string `csv:"Namespace"`
	Name      string `csv:"Name"`
	Transport string `csv:"TRANSPORT"`
	URL       string `csv:"URL"`
	Scopes    string `csv:"SCOPES"`
	Created   string `csv:"CREATED"`
	Age       string `csv:"AGE"`
}

// mcpServerDetailedOutput is used for JSON/YAML output.
type mcpServerDetailedOutput struct {
	Name      string               `json:"name"`
	Namespace string               `json:"namespace"`
	Spec      v1alpha1.MCPServerSpec `json:"spec"`
}

// PrintMCPServers prints all Dapr MCPServer resources.
func PrintMCPServers(name, namespace, outputFormat string) error {
	return writeMCPServers(os.Stdout, func() (*v1alpha1.MCPServerList, error) {
		client, err := DaprClient()
		if err != nil {
			return nil, err
		}

		return ListMCPServers(client, namespace)
	}, name, outputFormat)
}

// ListMCPServers lists MCPServer resources from Kubernetes.
func ListMCPServers(client versioned.Interface, namespace string) (*v1alpha1.MCPServerList, error) {
	list, err := client.MCPServerV1alpha1().MCPServers(namespace).List(meta_v1.ListOptions{})
	// This means that the Dapr MCPServer CRD is not installed and
	// therefore no MCPServer items exist.
	if apierrors.IsNotFound(err) {
		list = &v1alpha1.MCPServerList{
			Items: []v1alpha1.MCPServer{},
		}
	} else if err != nil {
		return nil, err
	}

	return list, nil
}

func writeMCPServers(writer io.Writer, getFunc func() (*v1alpha1.MCPServerList, error), name, outputFormat string) error {
	servers, err := getFunc()
	if err != nil {
		return err
	}

	filtered := []v1alpha1.MCPServer{}
	filteredSpecs := []mcpServerDetailedOutput{}
	for _, s := range servers.Items {
		serverName := s.GetName()
		if name == "" || strings.EqualFold(serverName, name) {
			filtered = append(filtered, s)
			filteredSpecs = append(filteredSpecs, mcpServerDetailedOutput{
				Name:      serverName,
				Namespace: s.GetNamespace(),
				Spec:      s.Spec,
			})
		}
	}

	if outputFormat == "" || outputFormat == "list" {
		return printMCPServerList(writer, filtered)
	}

	sort.Slice(filteredSpecs, func(i, j int) bool {
		return filteredSpecs[i].Namespace > filteredSpecs[j].Namespace
	})
	return utils.PrintDetail(writer, outputFormat, filteredSpecs)
}

func printMCPServerList(writer io.Writer, list []v1alpha1.MCPServer) error {
	out := []MCPServerOutput{}
	for _, s := range list {
		out = append(out, MCPServerOutput{
			Name:      s.GetName(),
			Namespace: s.GetNamespace(),
			Transport: mcpTransport(&s),
			URL:       mcpURL(&s),
			Created:   s.CreationTimestamp.Format("2006-01-02 15:04.05"),
			Age:       age.GetAge(s.CreationTimestamp.Time),
			Scopes:    strings.Join(s.Scopes, ","),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Namespace > out[j].Namespace
	})
	return utils.MarshalAndWriteTable(writer, out)
}

// mcpTransport returns the transport type string for the MCPServer.
func mcpTransport(s *v1alpha1.MCPServer) string {
	switch {
	case s.Spec.Endpoint.StreamableHTTP != nil:
		return "streamable_http"
	case s.Spec.Endpoint.SSE != nil:
		return "sse"
	case s.Spec.Endpoint.Stdio != nil:
		return "stdio"
	default:
		return ""
	}
}

// mcpURL returns the URL or command for the MCPServer.
func mcpURL(s *v1alpha1.MCPServer) string {
	switch {
	case s.Spec.Endpoint.StreamableHTTP != nil:
		return s.Spec.Endpoint.StreamableHTTP.URL
	case s.Spec.Endpoint.SSE != nil:
		return s.Spec.Endpoint.SSE.URL
	case s.Spec.Endpoint.Stdio != nil:
		return s.Spec.Endpoint.Stdio.Command
	default:
		return ""
	}
}
