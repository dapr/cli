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
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/dapr/cli/utils"
	"gopkg.in/yaml.v2"
)

const (
	credentialsContainerPath  = "/var/run/dapr/credentials"
	trustAnchorsContainerPath = credentialsContainerPath + "/" + trustAnchorsFile
	sentryAddressHostGateway  = "host.docker.internal:50001"
)

func sentryAddressForContainer(dockerNetwork string) string {
	if dockerNetwork != "" {
		return DaprSentryContainerName + ":50001"
	}
	return sentryAddressHostGateway
}

func appendDockerHostGateway(dockerRunArgs []string, dockerNetwork string) []string {
	if dockerNetwork == "" && runtime.GOOS != daprWindowsOS {
		return append(dockerRunArgs, "--add-host=host.docker.internal:host-gateway")
	}
	return dockerRunArgs
}

func appendMTLSCredentialsMount(dockerRunArgs []string, installDir string) []string {
	certsDir := GetDaprCertsPath(installDir)
	return append(dockerRunArgs, "-v", certsDir+":"+credentialsContainerPath)
}

func mtlsControlPlaneServiceArgs(dockerNetwork string) []string {
	return []string{
		"--mode", sentryStandaloneMode,
		"--tls-enabled",
		"--trust-domain", defaultTrustDomain,
		"--trust-anchors-file", trustAnchorsContainerPath,
		"--sentry-address", sentryAddressForContainer(dockerNetwork),
	}
}

func appendMTLSContainerRunArgs(dockerRunArgs []string, info initInfo) []string {
	if !info.enableMTLS {
		return dockerRunArgs
	}
	dockerRunArgs = appendMTLSCredentialsMount(dockerRunArgs, info.installDir)
	return appendDockerHostGateway(dockerRunArgs, info.dockerNetwork)
}

func buildSentryContainerRunArgs(info initInfo, image string) []string {
	certsDir := GetDaprCertsPath(info.installDir)
	configPath := GetDaprConfigPath(info.installDir)
	sentryContainerName := utils.CreateContainerName(DaprSentryContainerName, info.dockerNetwork)

	args := []string{
		"run",
		"--name", sentryContainerName,
		"--restart", "always",
		"-d",
		"--entrypoint", "./sentry",
		"-v", certsDir + ":/var/run/dapr/credentials",
		"-v", configPath + ":" + sentryConfigContainerPath + ":ro",
	}

	if info.dockerNetwork != "" {
		args = append(args,
			"--network", info.dockerNetwork,
			"--network-alias", DaprSentryContainerName)
	} else {
		args = append(args,
			"-p", fmt.Sprintf("%v:50001", sentryGRPCPort),
			"-p", fmt.Sprintf("%v:8080", sentryHealthPort),
			"-p", fmt.Sprintf("%v:9090", sentryMetricPort),
		)
	}

	args = append(args, image,
		"--mode", sentryStandaloneMode,
		"--config", sentryConfigContainerPath,
		"--issuer-credentials", credentialsContainerPath,
		"--trust-domain", defaultTrustDomain,
	)

	return args
}

func mergeMTLSIntoConfiguration(filePath string) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var config configuration
	if err := yaml.Unmarshal(b, &config); err != nil {
		return err
	}
	if config.APIVersion == "" {
		config.APIVersion = "dapr.io/v1alpha1"
	}
	if config.Kind == "" {
		config.Kind = "Configuration"
	}
	if config.Metadata.Name == "" {
		config.Metadata.Name = "daprConfig"
	}
	config.Spec.MTLS.Enabled = true

	out, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, out, 0o644)
}

func runParallelInitSteps(steps []func(*sync.WaitGroup, chan<- error, initInfo), info initInfo) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, len(steps))
	wg.Add(len(steps))
	for _, step := range steps {
		go step(&wg, errorChan, info)
	}
	go func() {
		wg.Wait()
		close(errorChan)
	}()
	for err := range errorChan {
		if err != nil {
			return err
		}
	}
	return nil
}
