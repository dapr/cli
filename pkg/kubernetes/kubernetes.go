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

package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/helm/pkg/strvals"

	"github.com/dapr/cli/pkg/print"
	cli_ver "github.com/dapr/cli/pkg/version"
	"github.com/dapr/cli/utils"
)

const (
	daprReleaseName = "dapr"
	daprHelmRepo    = "https://dapr.github.io/helm-charts"
	latestVersion   = "latest"
)

type InitConfiguration struct {
	Version                   string
	Namespace                 string
	EnableMTLS                bool
	EnableHA                  bool
	Args                      []string
	Wait                      bool
	Timeout                   uint
	ImageRegistryURI          string
	ImageVariant              string
	RootCertificateFilePath   string
	IssuerCertificateFilePath string
	IssuerPrivateKeyFilePath  string
}

// Init deploys the Dapr operator using the supplied runtime version.
func Init(config InitConfiguration) error {
	msg := "Deploying the Dapr control plane to your cluster..."

	stopSpinning := print.Spinner(os.Stdout, msg)
	defer stopSpinning(print.Failure)
	if err := install(config); err != nil {
		return err
	}

	stopSpinning(print.Success)

	return nil
}

func createNamespace(namespace string) error {
	_, client, err := GetKubeConfigClient()
	if err != nil {
		return fmt.Errorf("can't connect to a Kubernetes cluster: %w", err)
	}

	ns := &v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespace,
		},
	}
	// try to create the namespace if it doesn't exist. ok to ignore error.
	client.CoreV1().Namespaces().Create(context.TODO(), ns, meta_v1.CreateOptions{})
	return nil
}

func helmConfig(namespace string) (*helm.Configuration, error) {
	ac := helm.Configuration{}
	flags := &genericclioptions.ConfigFlags{
		Namespace: &namespace,
	}
	err := ac.Init(flags, namespace, "secret", debugLogf)
	return &ac, err
}

func getVersion(version string) (string, error) {
	if version == latestVersion {
		var err error
		version, err = cli_ver.GetDaprVersion()
		if err != nil {
			return "", fmt.Errorf("cannot get the latest release version: %w", err)
		}
		version = strings.TrimPrefix(version, "v")
	}
	return version, nil
}

func createTempDir() (string, error) {
	dir, err := os.MkdirTemp("", "dapr")
	if err != nil {
		return "", fmt.Errorf("error creating temp dir: %w", err)
	}
	return dir, nil
}

func locateChartFile(dirPath string) (string, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(dirPath, files[0].Name()), nil
}

func daprChart(version string, config *helm.Configuration) (*chart.Chart, error) {
	pull := helm.NewPullWithOpts(helm.WithConfig(config))
	pull.RepoURL = utils.GetEnv("DAPR_HELM_REPO_URL", daprHelmRepo)
	pull.Username = utils.GetEnv("DAPR_HELM_REPO_USERNAME", "")
	pull.Password = utils.GetEnv("DAPR_HELM_REPO_PASSWORD", "")

	pull.Settings = &cli.EnvSettings{}

	if version != latestVersion {
		pull.Version = chartVersion(version)
	}

	dir, err := createTempDir()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	pull.DestDir = dir

	_, err = pull.Run(daprReleaseName)
	if err != nil {
		return nil, err
	}

	chartPath, err := locateChartFile(dir)
	if err != nil {
		return nil, err
	}
	return loader.Load(chartPath)
}

func chartValues(config InitConfiguration, version string) (map[string]interface{}, error) {
	chartVals := map[string]interface{}{}
	err := utils.ValidateImageVariant(config.ImageVariant)
	if err != nil {
		return nil, err
	}
	helmVals := []string{
		fmt.Sprintf("global.ha.enabled=%t", config.EnableHA),
		fmt.Sprintf("global.mtls.enabled=%t", config.EnableMTLS),
		fmt.Sprintf("global.tag=%s", utils.GetVariantVersion(version, config.ImageVariant)),
	}
	if len(config.ImageRegistryURI) != 0 {
		helmVals = append(helmVals, fmt.Sprintf("global.registry=%s", config.ImageRegistryURI))
	}
	helmVals = append(helmVals, config.Args...)

	if config.RootCertificateFilePath != "" && config.IssuerCertificateFilePath != "" && config.IssuerPrivateKeyFilePath != "" {
		rootCertBytes, issuerCertBytes, issuerKeyBytes, err := parseCertificateFiles(
			config.RootCertificateFilePath,
			config.IssuerCertificateFilePath,
			config.IssuerPrivateKeyFilePath,
		)
		if err != nil {
			return nil, err
		}
		helmVals = append(helmVals, fmt.Sprintf("dapr_sentry.tls.root.certPEM=%s", string(rootCertBytes)),
			fmt.Sprintf("dapr_sentry.tls.issuer.certPEM=%s", string(issuerCertBytes)),
			fmt.Sprintf("dapr_sentry.tls.issuer.keyPEM=%s", string(issuerKeyBytes)),
		)
	}

	for _, v := range helmVals {
		if err := strvals.ParseInto(v, chartVals); err != nil {
			return nil, err
		}
	}
	return chartVals, nil
}

func install(config InitConfiguration) error {
	err := createNamespace(config.Namespace)
	if err != nil {
		return err
	}

	helmConf, err := helmConfig(config.Namespace)
	if err != nil {
		return err
	}

	daprChart, err := daprChart(config.Version, helmConf)
	if err != nil {
		return err
	}

	version, err := getVersion(config.Version)
	if err != nil {
		return err
	}

	err = applyCRDs(fmt.Sprintf("v%s", version))
	if err != nil {
		return err
	}

	installClient := helm.NewInstall(helmConf)
	installClient.ReleaseName = daprReleaseName
	installClient.Namespace = config.Namespace
	installClient.Wait = config.Wait
	installClient.Timeout = time.Duration(config.Timeout) * time.Second

	values, err := chartValues(config, version)
	if err != nil {
		return err
	}

	if _, err = installClient.Run(daprChart, values); err != nil {
		return err
	}
	return nil
}

func debugLogf(format string, v ...interface{}) {
}

func confirmExist(cfg *helm.Configuration) (bool, error) {
	client := helm.NewGet(cfg)
	release, err := client.Run(daprReleaseName)

	if release == nil {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
