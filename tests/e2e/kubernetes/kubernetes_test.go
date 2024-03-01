//go:build e2e
// +build e2e

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

package kubernetes_test

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dapr/cli/tests/e2e/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKubernetesNonHAModeMTLSDisabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           false,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesHAModeMTLSDisabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             true,
		MTLSEnabled:           false,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesDev(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		DevEnabled:            true,
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		DevEnabled:   true,
		UninstallAll: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  false,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesNonHAModeMTLSEnabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesHAModeMTLSEnabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             true,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		UninstallAll: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  false,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesInitWithCustomCert(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		InitWithCustomCert:    true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

// Test for certificate renewal

func TestRenewCertificateMTLSEnabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// tests for certifcate renewal.
	tests = append(tests, common.GetTestForCertRenewal(currentVersionDetails, installOpts)...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestRenewCertificateMTLSDisabled(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           false,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// tests for certifcate renewal.
	tests = append(tests, common.GetTestForCertRenewal(currentVersionDetails, installOpts)...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestRenewCertWithPrivateKey(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// tests for certifcate renewal with newly generated certificates when pem encoded private root.key file is provided
	tests = append(tests, []common.TestCase{
		{"Renew certificate which expires in less than 30 days", common.UseProvidedPrivateKeyAndRenewCerts(currentVersionDetails, installOpts)},
	}...)

	tests = append(tests, common.GetTestsPostCertificateRenewal(currentVersionDetails, installOpts)...)
	tests = append(tests, []common.TestCase{
		{"Cert Expiry warning message check " + currentVersionDetails.RuntimeVersion, common.CheckMTLSStatus(currentVersionDetails, installOpts, true)},
	}...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesUninstall(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)
	// setup tests
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestRenewCertWithIncorrectFlags(t *testing.T) {
	common.EnsureUninstall(true, true)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// tests for certifcate renewal with incorrect set of flags provided.
	tests = append(tests, []common.TestCase{
		{"Renew certificate with incorrect flags", common.NegativeScenarioForCertRenew()},
	}...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

// install dapr control plane with mariner docker images.
// Renew the certificate of this control plane.
func TestK8sInstallwithMarinerImagesAndRenewCertificate(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	//	install with mariner images
	currentVersionDetails.ImageVariant = "mariner"

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// tests for certifcate renewal.
	tests = append(tests, common.GetTestForCertRenewal(currentVersionDetails, installOpts)...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesInstallwithoutRuntimeVersionFlag(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, true)

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestK8sInstallwithoutRuntimeVersionwithMarinerImagesFlag(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, true)

	//	install with mariner images
	currentVersionDetails.ImageVariant = "mariner"

	tests := []common.TestCase{}
	installOpts := common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           true,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	}

	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, installOpts)...)

	// teardown everything
	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func TestKubernetesLocalFileHelmRepoInstall(t *testing.T) {
	// ensure clean env for test
	ensureCleanEnv(t, false)

	// create a temp dir to store the helm repo
	helmRepoPath, err := os.MkdirTemp("", "dapr-e2e-kube-with-env-*")
	assert.NoError(t, err)
	// defer os.RemoveAll(helmRepoPath) // clean up

	// copy all .tar.gz files from testdata dir and uncompress them
	copyAndUncompressTarGzFiles(t, helmRepoPath)

	// point the env var to the dir containing both dapr and dapr-dashboard helm charts
	t.Setenv("DAPR_HELM_REPO_URL", helmRepoPath)

	// setup tests
	tests := []common.TestCase{}
	tests = append(tests, common.GetTestsOnInstall(currentVersionDetails, common.TestOptions{
		HAEnabled:             false,
		MTLSEnabled:           false,
		ApplyComponentChanges: true,
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        true,
			common.ClusterRoleBindings: true,
		},
	})...)

	tests = append(tests, common.GetTestsOnUninstall(currentVersionDetails, common.TestOptions{
		CheckResourceExists: map[common.Resource]bool{
			common.CustomResourceDefs:  true,
			common.ClusterRoles:        false,
			common.ClusterRoleBindings: false,
		},
	})...)

	// execute tests
	for _, tc := range tests {
		t.Run(tc.Name, tc.Callable)
	}
}

func copyAndUncompressTarGzFiles(t *testing.T, destination string) {
	// find all .tar.gz files in testdata dir
	files, err := filepath.Glob(filepath.Join("testdata", "*.tgz"))
	require.NoError(t, err)

	for _, file := range files {
		// untar the dapr/dashboard helm .tar.gz, get back the root dir of the untarred files
		// it's either 'dapr' or 'dapr-dashboard'
		rootDir, err := untarDaprHelmGzFile(file, destination)
		require.NoError(t, err)

		// rename the root dir to the base name of the .tar.gz file
		// (eg. /var/folders/4s/w0gdrc957k11vbkgyhjrk12w0000gn/T/dapr-e2e-kube-with-env-404115459/dapr-1.12.0)
		base := filepath.Base(strings.TrimSuffix(file, filepath.Ext(file)))
		err = os.Rename(filepath.Join(destination, rootDir), filepath.Join(destination, base))
		require.NoError(t, err)
	}
}

func untarDaprHelmGzFile(file string, destination string) (string, error) {
	// open the tar.gz file
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// create a gzip reader
	gr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gr.Close()

	// create a tar reader
	tr := tar.NewReader(gr)

	rootDir := ""
	// iterate through all the files in the tarball
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // end of tarball
		}
		if err != nil {
			return "", err
		}

		// build the full destination path
		filename := filepath.Join(destination, hdr.Name)

		// ensure the destination directory exists
		dir := filepath.Dir(filename)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0700); err != nil {
				return "", err
			}
		}

		// the root dir for all files is the same
		rootDir = strings.FieldsFunc(hdr.Name,
			func(c rune) bool {
				return os.PathSeparator == c
			})[0]

		// create the destination file
		dstFile, err := os.Create(filename)
		if err != nil {
			return "", err
		}
		defer dstFile.Close()

		// copy the file contents
		if _, err := io.Copy(dstFile, tr); err != nil {
			return "", err
		}
	}

	return rootDir, nil
}
