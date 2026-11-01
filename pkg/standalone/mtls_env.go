package standalone

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvDaprTrustAnchors = "DAPR_TRUST_ANCHORS"
	EnvDaprCertChain    = "DAPR_CERT_CHAIN"
	EnvDaprCertKey      = "DAPR_CERT_KEY"
	EnvNamespace        = "NAMESPACE"

	defaultNamespace = "default"
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
			return nil,  fmt.Errorf("mTLS file doesn't exists: file %q: %w", c.file, err)
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
		if errors.Is(err, fs.ErrNotExist) {
			return false
		}
	}

	return true
}
