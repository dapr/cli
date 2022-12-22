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
		ghcrImageName:      redisGhcrImageName,
		dockerHubImageName: redisDockerImageName,
		imageRegistryURL:   "",
		imageRegistryName:  "ghcr",
	}
	zipkinImageInfo := daprImageInfo{
		ghcrImageName:      zipkinGhcrImageName,
		dockerHubImageName: zipkinDockerImageName,
		imageRegistryURL:   "",
		imageRegistryName:  "ghcr",
	}
	placementImageInfo := daprImageInfo{
		ghcrImageName:      daprGhcrImageName,
		dockerHubImageName: daprDockerImageName,
		imageRegistryURL:   "",
		imageRegistryName:  "ghcr",
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
	expectedRedisImageName := "redis:6"
	expectedZipkinImageName := "openzipkin/zipkin"
	expectedPlacementImageName := "daprio/dapr"

	redisImageInfo := daprImageInfo{
		ghcrImageName:      redisGhcrImageName,
		dockerHubImageName: redisDockerImageName,
		imageRegistryURL:   "",
		imageRegistryName:  "dockerhub",
	}
	zipkinImageInfo := daprImageInfo{
		ghcrImageName:      zipkinGhcrImageName,
		dockerHubImageName: zipkinDockerImageName,
		imageRegistryURL:   "",
		imageRegistryName:  "dockerhub",
	}
	placementImageInfo := daprImageInfo{
		ghcrImageName:      daprGhcrImageName,
		dockerHubImageName: daprDockerImageName,
		imageRegistryURL:   "",
		imageRegistryName:  "dockerhub",
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
		ghcrImageName:      redisGhcrImageName,
		dockerHubImageName: redisDockerImageName,
		imageRegistryURL:   "docker.io/username",
		imageRegistryName:  "",
	}
	zipkinImageInfo := daprImageInfo{
		ghcrImageName:      zipkinGhcrImageName,
		dockerHubImageName: zipkinDockerImageName,
		imageRegistryURL:   "docker.io/username",
		imageRegistryName:  "",
	}
	placementImageInfo := daprImageInfo{
		ghcrImageName:      daprGhcrImageName,
		dockerHubImageName: daprDockerImageName,
		imageRegistryURL:   "docker.io/username",
		imageRegistryName:  "",
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
	redisImageInfo := daprImageInfo{
		ghcrImageName:      redisGhcrImageName,
		dockerHubImageName: redisDockerImageName,
		imageRegistryURL:   "docker.io",
		imageRegistryName:  "",
	}
	zipkinImageInfo := daprImageInfo{
		ghcrImageName:      zipkinGhcrImageName,
		dockerHubImageName: zipkinDockerImageName,
		imageRegistryURL:   ghcrURI,
		imageRegistryName:  "",
	}
	placementImageInfo := daprImageInfo{
		ghcrImageName:      daprGhcrImageName,
		dockerHubImageName: daprDockerImageName,
		imageRegistryURL:   "",
		imageRegistryName:  "value_other_than_dockerhub_or_ghcr",
	}

	tests := []struct {
		name      string
		args      daprImageInfo
		expect    string
		expectErr bool
	}{
		{"Test Redis image name", redisImageInfo, "", true},
		{"Test Zipkin image name", zipkinImageInfo, "", true},
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

func TestCheckFallbackImg(t *testing.T) {
	daprImgWithPrivateRegAndDefAsDocker := daprImageInfo{
		ghcrImageName:      daprGhcrImageName,
		dockerHubImageName: daprDockerImageName,
		imageRegistryURL:   "example.io/user",
		imageRegistryName:  "dockerhub",
	}
	daprImgWithPrivateRegAndDefAsGHCR := daprImageInfo{
		ghcrImageName:      daprGhcrImageName,
		dockerHubImageName: daprDockerImageName,
		imageRegistryURL:   "example.io/user",
		imageRegistryName:  "ghcr",
	}
	daprImgWithPrivateRegAndNoDef := daprImageInfo{
		ghcrImageName:      daprGhcrImageName,
		dockerHubImageName: daprDockerImageName,
		imageRegistryURL:   "example.io/user",
		imageRegistryName:  "",
	}
	daprImgWithDefAsDocker := daprImageInfo{
		ghcrImageName:      daprGhcrImageName,
		dockerHubImageName: daprDockerImageName,
		imageRegistryURL:   "",
		imageRegistryName:  "dockerhub",
	}
	daprImgWithDefAsGHCR := daprImageInfo{
		ghcrImageName:      daprGhcrImageName,
		dockerHubImageName: daprDockerImageName,
		imageRegistryURL:   "",
		imageRegistryName:  "ghcr",
	}

	tests := []struct {
		name      string
		imageInfo daprImageInfo
		fromDir   string
		expect    bool
	}{
		{"checkFallbackImg() with private registry and def as Docker Hub", daprImgWithPrivateRegAndDefAsDocker, "", false},
		{"checkFallbackImg() with private registry and def as GHCR", daprImgWithPrivateRegAndDefAsGHCR, "", false},
		{"checkFallbackImg() with private registry with no Def", daprImgWithPrivateRegAndNoDef, "", false},
		{"checkFallbackImg() with no private registry and def as Docker Hub", daprImgWithDefAsDocker, "", false},
		{"checkFallbackImg() with no private registry and def as GHCR", daprImgWithDefAsGHCR, "", true},
		{"checkFallbackImg() airgap mode with no private registry and def as GHCR", daprImgWithDefAsGHCR, "testDir", false},
		{"checkFallbackImg() airgap mode with no private registry and def as Docker Hub", daprImgWithDefAsDocker, "testDir", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := useGHCR(test.imageInfo, test.fromDir)
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

func TestNonDefaultInstallWithFlag(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "daprtest-nondefault-install-withflag-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir) // clean up.

	err = Init(latestVersion, latestVersion, "", true, "", "", "docker", "", tmpDir)
	assert.NoError(t, err)

	daprCMD := binaryFilePathWithDir(daprBinPath(tmpDir), "daprd")

	assert.FileExists(t, daprCMD)
}

func TestNonDefaultInstallWithEnvVar(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "daprtest-nondefault-install-withenv-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir) // clean up.

	t.Setenv("DAPR_PATH", tmpDir)
	defer os.Unsetenv("DAPR_PATH")

	err = Init(latestVersion, latestVersion, "", true, "", "", "docker", "", "")
	assert.NoError(t, err)

	daprCMD := binaryFilePathWithDir(daprBinPath(tmpDir), "daprd")

	assert.FileExists(t, daprCMD)
}

func TestNonDefaultInstallWithFlagAndEnvVar(t *testing.T) {
	tmpDir1, err := os.MkdirTemp("", "daprtest-nondefault-install-withboth-flag-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir1) // clean up.

	tmpDir2, err := os.MkdirTemp("", "daprtest-nondefault-install-withboth-env-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir2) // clean up.

	t.Setenv("DAPR_PATH", tmpDir2)
	defer os.Unsetenv("DAPR_PATH")

	err = Init(latestVersion, latestVersion, "", true, "", "", "docker", "", tmpDir1)
	assert.NoError(t, err)

	daprCMD := binaryFilePathWithDir(daprBinPath(tmpDir1), "daprd")

	assert.FileExists(t, daprCMD)

	daprCMD = binaryFilePathWithDir(daprBinPath(tmpDir2), "daprd")

	assert.NoFileExists(t, daprCMD)
}
