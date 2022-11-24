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

package standalone

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStandaloneConfig(t *testing.T) {
	testFile := "./test.yaml"

	t.Run("Standalone config", func(t *testing.T) {
		expectConfigZipkin := `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec:
  tracing:
    samplingRate: "1"
    zipkin:
      endpointAddress: http://test_zipkin_host:9411/api/v2/spans
`
		os.Remove(testFile)
		createDefaultConfiguration("test_zipkin_host", testFile)
		assert.FileExists(t, testFile)
		content, err := os.ReadFile(testFile)
		assert.NoError(t, err)
		assert.Equal(t, expectConfigZipkin, string(content))
	})

	t.Run("Standalone config slim", func(t *testing.T) {
		expectConfigSlim := `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec: {}
`
		os.Remove(testFile)
		createDefaultConfiguration("", testFile)
		assert.FileExists(t, testFile)
		content, err := os.ReadFile(testFile)
		assert.NoError(t, err)
		assert.Equal(t, expectConfigSlim, string(content))
	})

	os.Remove(testFile)
}

func TestResolveImageWithGHCR(t *testing.T) {
	expectedRedisImageName := "ghcr.io/dapr/3rdparty/redis:6"
	expectedZipkinImageName := "ghcr.io/dapr/3rdparty/zipkin"
	expectedPlacementImageName := "ghcr.io/dapr/dapr"

	redisImageInfo := daprImageInfo{
		ghcrImagePath:        redisGhcrImagePath,
		dockerHubImagePath:   redisDockerHubImagePath,
		customRegistryURL:    "",
		containerRegistryURL: "ghcr.io",
	}
	zipkinImageInfo := daprImageInfo{
		ghcrImagePath:        zipkinGhcrImagePath,
		dockerHubImagePath:   zipkinDockerHubImagePath,
		customRegistryURL:    "",
		containerRegistryURL: "ghcr.io",
	}
	placementImageInfo := daprImageInfo{
		ghcrImagePath:        daprGhcrImagePath,
		dockerHubImagePath:   daprDockerHubImagePath,
		customRegistryURL:    "",
		containerRegistryURL: "ghcr.io",
	}

	tests := []struct {
		name      string
		args      daprImageInfo
		expect    string
		expectErr bool
	}{
		{"Test Redis image name", redisImageInfo, expectedRedisImageName, false},
		{"Test Zipkin image name", zipkinImageInfo, expectedZipkinImageName, false},
		{"Test Dapr image name", placementImageInfo, expectedPlacementImageName, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveImageURI(test.args)
			assert.Equal(t, test.expectErr, err != nil)
			assert.Equal(t, test.expect, got)
		})
	}
}

func TestResolveImageWithDockerHub(t *testing.T) {
	expectedRedisImageName := "docker.io/redis:6"
	expectedZipkinImageName := "docker.io/openzipkin/zipkin"
	expectedPlacementImageName := "docker.io/daprio/dapr"

	redisImageInfo := daprImageInfo{
		ghcrImagePath:        redisGhcrImagePath,
		dockerHubImagePath:   redisDockerHubImagePath,
		customRegistryURL:    "",
		containerRegistryURL: "docker.io",
	}
	zipkinImageInfo := daprImageInfo{
		ghcrImagePath:        zipkinGhcrImagePath,
		dockerHubImagePath:   zipkinDockerHubImagePath,
		customRegistryURL:    "",
		containerRegistryURL: "docker.io",
	}
	placementImageInfo := daprImageInfo{
		ghcrImagePath:        daprGhcrImagePath,
		dockerHubImagePath:   daprDockerHubImagePath,
		customRegistryURL:    "",
		containerRegistryURL: "docker.io",
	}

	tests := []struct {
		name      string
		args      daprImageInfo
		expect    string
		expectErr bool
	}{
		{"Test Redis image name", redisImageInfo, expectedRedisImageName, false},
		{"Test Zipkin image name", zipkinImageInfo, expectedZipkinImageName, false},
		{"Test Dapr image name", placementImageInfo, expectedPlacementImageName, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveImageURI(test.args)
			assert.Equal(t, test.expectErr, err != nil)
			assert.Equal(t, test.expect, got)
		})
	}
}

func TestResolveImageWithPrivateRegistry(t *testing.T) {
	expectedRedisImageName := "docker.io/username/dapr/3rdparty/redis:6"
	expectedZipkinImageName := "docker.io/username/dapr/3rdparty/zipkin"
	expectedPlacementImageName := "docker.io/username/dapr/dapr"

	redisImageInfo := daprImageInfo{
		ghcrImagePath:        redisGhcrImagePath,
		dockerHubImagePath:   redisDockerHubImagePath,
		customRegistryURL:    "docker.io/username",
		containerRegistryURL: "",
	}
	zipkinImageInfo := daprImageInfo{
		ghcrImagePath:        zipkinGhcrImagePath,
		dockerHubImagePath:   zipkinDockerHubImagePath,
		customRegistryURL:    "docker.io/username",
		containerRegistryURL: "",
	}
	placementImageInfo := daprImageInfo{
		ghcrImagePath:        daprGhcrImagePath,
		dockerHubImagePath:   daprDockerHubImagePath,
		customRegistryURL:    "docker.io/username",
		containerRegistryURL: "",
	}

	tests := []struct {
		name      string
		args      daprImageInfo
		expect    string
		expectErr bool
	}{
		{"Test Redis image name", redisImageInfo, expectedRedisImageName, false},
		{"Test Zipkin image name", zipkinImageInfo, expectedZipkinImageName, false},
		{"Test Dapr image name", placementImageInfo, expectedPlacementImageName, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveImageURI(test.args)
			assert.Equal(t, test.expectErr, err != nil)
			assert.Equal(t, test.expect, got)
		})
	}
}

func TestResolveImageErr(t *testing.T) {
	placementImageInfo := daprImageInfo{
		ghcrImagePath:        daprGhcrImagePath,
		dockerHubImagePath:   daprDockerHubImagePath,
		customRegistryURL:    "",
		containerRegistryURL: "value_other_than_dockerhub_or_ghcr",
	}

	tests := []struct {
		name      string
		args      daprImageInfo
		expect    string
		expectErr bool
	}{
		{"Test Dapr image name", placementImageInfo, "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveImageURI(test.args)
			assert.Equal(t, test.expectErr, err != nil)
			assert.Equal(t, test.expect, got)
		})
	}
}

func TestIsAirGapInit(t *testing.T) {
	tests := []struct {
		name    string
		fromDir string
		expect  bool
	}{
		{"empty string", "", false},
		{"string with spaces", "   ", false},
		{"string with value", "./local-dir", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setAirGapInit(test.fromDir)
			assert.Equal(t, test.expect, isAirGapInit)
		})
	}
}
