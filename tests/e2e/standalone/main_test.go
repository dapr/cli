//go:build e2e || template

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

package standalone_test

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMain installs Dapr once for the entire test binary, removing the
// need for every test to call cmdUninstall/ensureDaprInstallation.
// Tests that need to test the install/uninstall lifecycle itself must
// reinstall Dapr in their t.Cleanup so subsequent tests still work.
func TestMain(m *testing.M) {
	// Start from a clean slate.
	cmdUninstall()

	if err := installDapr(); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: failed to install Dapr: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	cmdUninstall()
	os.Exit(code)
}

// installDapr performs a Dapr init for the test binary. This mirrors
// ensureDaprInstallation but does not require a *testing.T.
func installDapr() error {
	daprRuntimeVersion, ok := os.LookupEnv("DAPR_RUNTIME_PINNED_VERSION")
	if !ok {
		return fmt.Errorf("env var DAPR_RUNTIME_PINNED_VERSION not set")
	}
	daprDashboardVersion, ok := os.LookupEnv("DAPR_DASHBOARD_PINNED_VERSION")
	if !ok {
		return fmt.Errorf("env var DAPR_DASHBOARD_PINNED_VERSION not set")
	}

	if !isSlimMode() {
		if err := waitForPortsFreeDirect(60*time.Second, 58080, 58081, 50005); err != nil {
			return fmt.Errorf("waiting for container ports: %w", err)
		}
	}

	args := []string{
		"--runtime-version", daprRuntimeVersion,
		"--dashboard-version", daprDashboardVersion,
	}
	output, err := cmdInit(args...)
	if err != nil {
		return fmt.Errorf("dapr init: %s: %w", output, err)
	}

	if isSlimMode() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home dir: %w", err)
		}
		if err := createSlimComponents(filepath.Join(homeDir, ".dapr", "components")); err != nil {
			return fmt.Errorf("creating slim components: %w", err)
		}
	}

	return nil
}

// waitForPortsFreeDirect is a non-test variant of waitForPortsFree for
// use in TestMain where *testing.T is not available.
func waitForPortsFreeDirect(timeout time.Duration, ports ...int) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		allFree := true
		for _, port := range ports {
			ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				allFree = false
				break
			}
			ln.Close()
		}
		if allFree {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("ports %v not free within %v", ports, timeout)
}
