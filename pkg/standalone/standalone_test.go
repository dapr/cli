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
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dapr/cli/utils"
	"github.com/dapr/kit/ptr"
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
			t.Parallel()
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
			t.Parallel()
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
			t.Parallel()
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

func TestInitLogActualContainerRuntimeName(t *testing.T) {
	tests := []struct {
		containerRuntime string
		testName         string
	}{
		{"podman", "Init should log podman as container runtime"},
		{"docker", "Init should log docker as container runtime"},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			containerRuntimeAvailable := utils.IsContainerRuntimeInstalled(test.containerRuntime)
			if containerRuntimeAvailable {
				t.Skip("Skipping test as container runtime is available")
			}

			err := Init(latestVersion, latestVersion, "", false, "", "", test.containerRuntime, "", "", nil, ptr.Of("localhost:50006"), 0, 0, 0)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.containerRuntime)
		})
	}
}

func TestIsSchedulerIncluded(t *testing.T) {
	scenarios := []struct {
		version    string
		isIncluded bool
	}{
		{"1.13.0-rc.1", false},
		{"1.13.0", false},
		{"1.13.1", false},
		{"1.14.0", true},
		{"1.14.0-rc.1", true},
		{"1.14.0-mycompany.1", true},
		{"1.14.1", true},
	}
	for _, scenario := range scenarios {
		t.Run("isSchedulerIncludedIn"+scenario.version, func(t *testing.T) {
			included, err := isSchedulerIncluded(scenario.version)
			assert.NoError(t, err)
			assert.Equal(t, scenario.isIncluded, included)
		})
	}
}

func TestGetHostPorts(t *testing.T) {
	t.Run("getPlacementHostPort uses override when set", func(t *testing.T) {
		info := initInfo{placementHostPort: 7777}
		assert.Equal(t, 7777, getPlacementHostPort(info))
	})

	t.Run("getPlacementHostPort uses default when override is zero", func(t *testing.T) {
		info := initInfo{placementHostPort: 0}
		port := getPlacementHostPort(info)
		assert.Condition(t, func() bool { return port == 50005 || port == 6050 },
			"expected 50005 or 6050, got %d", port)
	})

	t.Run("getRedisHostPort uses override when set", func(t *testing.T) {
		info := initInfo{redisHostPort: 6380}
		assert.Equal(t, 6380, getRedisHostPort(info))
	})

	t.Run("getRedisHostPort uses default 6379 when override is zero", func(t *testing.T) {
		info := initInfo{redisHostPort: 0}
		assert.Equal(t, 6379, getRedisHostPort(info))
	})

	t.Run("getZipkinHostPort uses override when set", func(t *testing.T) {
		info := initInfo{zipkinHostPort: 9412}
		assert.Equal(t, 9412, getZipkinHostPort(info))
	})

	t.Run("getZipkinHostPort uses default 9411 when override is zero", func(t *testing.T) {
		info := initInfo{zipkinHostPort: 0}
		assert.Equal(t, 9411, getZipkinHostPort(info))
	})
}

func TestParseNetshExcludedRanges(t *testing.T) {
	t.Run("parses well-formed netsh output", func(t *testing.T) {
		output := `
Protocol tcp Port Exclusion Ranges

Start Port    End Port
----------    --------
      1024        1123
      2375        2375
     50000       50059

* - Administered port exclusions.
`
		ranges := parseNetshExcludedRanges(output)
		assert.Len(t, ranges, 3)
		assert.Equal(t, [2]int{1024, 1123}, ranges[0])
		assert.Equal(t, [2]int{2375, 2375}, ranges[1])
		assert.Equal(t, [2]int{50000, 50059}, ranges[2])
	})

	t.Run("returns nil for empty output", func(t *testing.T) {
		ranges := parseNetshExcludedRanges("")
		assert.Nil(t, ranges)
	})

	t.Run("skips non-numeric header lines", func(t *testing.T) {
		output := "Start Port    End Port\n----------    --------\n      6379        6379\n"
		ranges := parseNetshExcludedRanges(output)
		assert.Len(t, ranges, 1)
		assert.Equal(t, [2]int{6379, 6379}, ranges[0])
	})
}

func TestCheckPortAvailableOnCurrentPlatform(t *testing.T) {
	t.Run("available port returns nil", func(t *testing.T) {
		// Use a port that is virtually guaranteed to be free in CI.
		err := checkPortAvailable(0, "test service", "test-flag")
		// Port 0 is special; net.Listen picks an ephemeral port, so this should succeed.
		assert.NoError(t, err)
	})
}

func TestParseContainerRuntimeError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		assert.NoError(t, parseContainerRuntimeError("svc", nil))
	})

	t.Run("port is already allocated is wrapped with helpful message", func(t *testing.T) {
		err := fmt.Errorf("Error response from daemon: driver failed programming external connectivity: Bind for 0.0.0.0:6379 failed: port is already allocated")
		result := parseContainerRuntimeError("Redis state store", err)
		assert.Error(t, result)
		assert.Contains(t, result.Error(), "required port is already in use")
		assert.Contains(t, result.Error(), "--*-host-port")
	})

	t.Run("address already in use is wrapped with helpful message", func(t *testing.T) {
		err := fmt.Errorf("listen tcp 0.0.0.0:50005: bind: address already in use")
		result := parseContainerRuntimeError("placement service", err)
		assert.Error(t, result)
		assert.Contains(t, result.Error(), "required port is already in use")
	})

	t.Run("unrelated error passes through unchanged", func(t *testing.T) {
		err := fmt.Errorf("some other docker error")
		result := parseContainerRuntimeError("svc", err)
		assert.Equal(t, err, result)
	})
}
