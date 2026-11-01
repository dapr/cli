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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	EnvDaprTrustAnchors = "DAPR_TRUST_ANCHORS"
	EnvDaprCertChain    = "DAPR_CERT_CHAIN"
	EnvDaprCertKey      = "DAPR_CERT_KEY"
	EnvNamespace        = "NAMESPACE"

	defaultNamespace = "default"

	sentryTrustDomain = "cluster.local"
	sentryPort        = 50001
	sentryWindowsPort = 6010
)

var mtlsCertFiles = [...]struct{ env, file string }{
	{EnvDaprTrustAnchors, "ca.crt"},
	{EnvDaprCertChain, "issuer.crt"},
	{EnvDaprCertKey, "issuer.key"},
}

func MTLSEnv(daprDir string) (map[string]string, error) {
	certsDir := GetDaprSentryCertsPath(daprDir)

	out := make(map[string]string, len(mtlsCertFiles)+1)
	for _, c := range mtlsCertFiles {
		b, err := os.ReadFile(filepath.Join(certsDir, c.file))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return nil, fmt.Errorf("reading mTLS file %q: %w", c.file, err)
		}
		out[c.env] = strings.TrimSpace(string(b))
	}

	return out, nil
}

func EnsureCerts(certsDir string) bool {
	for _, c := range mtlsCertFiles {
		_, err := os.ReadFile(filepath.Join(certsDir, c.file))
		if err != nil {
			return false
		}
	}

	return true
}

// sentryHostPort is the host port the local Sentry service is published on.
func sentryHostPort() int {
	if runtime.GOOS == daprWindowsOS {
		return sentryWindowsPort
	}
	return sentryPort
}

// SentryAddress is the address daprd uses to reach the local Sentry service.
func SentryAddress() string {
	return fmt.Sprintf("localhost:%d", sentryHostPort())
}
