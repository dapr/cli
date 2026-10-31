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

func MTLSEnv(daprDir string, lookupEnv func(string) (string, bool)) (map[string]string, error) {
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	certsDir := GetDaprSentryCertsPath(daprDir)

	out := make(map[string]string, len(mtlsCertFiles)+1)
	for _, c := range mtlsCertFiles {
		if _, set := lookupEnv(c.env); set {
			continue // caller's environment wins; don't read the file
		}
		b, err := os.ReadFile(filepath.Join(certsDir, c.file))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return nil, nil // certs not generated: mTLS not set up
		case err != nil:
			return nil, fmt.Errorf("reading mTLS file %q: %w", c.file, err)
		}
		out[c.env] = strings.TrimSpace(string(b))
	}

	if len(out) == 0 {
		return nil, nil // everything already exported
	}
	if _, set := lookupEnv(EnvNamespace); !set {
		out[EnvNamespace] = defaultNamespace
	}
	return out, nil
}
