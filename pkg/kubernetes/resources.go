package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/dapr/cli/pkg/print"
)

func getResources(resourcesFolder string) ([]client.Object, error) {
	// Create YAML decoder
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	// Read files from the resources folder
	files, err := os.ReadDir(resourcesFolder)
	if err != nil {
		return nil, fmt.Errorf("error reading resources folder: %w", err)
	}

	var resources []client.Object
	for _, file := range files {
		if file.IsDir() || (!strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".json")) {
			continue
		}

		// Read file content
		content, err := os.ReadFile(filepath.Join(resourcesFolder, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("error reading file %s: %w", file.Name(), err)
		}

		// Decode YAML/JSON to unstructured
		obj := &unstructured.Unstructured{}
		_, _, err = decUnstructured.Decode(content, nil, obj)
		if err != nil {
			return nil, fmt.Errorf("error decoding file %s: %w", file.Name(), err)
		}

		resources = append(resources, obj)
	}

	return resources, nil
}

func createOrUpdateResources(ctx context.Context, cl client.Client, resources []client.Object, namespace string) error {
	// create resources in k8s
	for _, resource := range resources {
		// clone the resource to avoid modifying the original
		obj := resource.DeepCopyObject().(*unstructured.Unstructured)
		// Set namespace on the resource metadata
		obj.SetNamespace(namespace)

		print.InfoStatusEvent(os.Stdout, "Deploying resource %q kind %q to Kubernetes", obj.GetName(), obj.GetKind())

		if err := cl.Create(ctx, obj); err != nil {
			if k8serrors.IsAlreadyExists(err) {
				print.InfoStatusEvent(os.Stdout, "Resource %q kind %q already exists, updating", obj.GetName(), obj.GetKind())
				if err := cl.Update(ctx, obj); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("error deploying resource %q kind %q to Kubernetes: %w", obj.GetName(), obj.GetKind(), err)
			}
		}
	}
	return nil
}
