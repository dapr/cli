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

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
)

var (
	injectTargetResource          string
	injectTargetNamespace         string
	injectAppID                   string
	injectAppPort                 int
	injectConfig                  string
	injectAppProtocol             string
	injectEnableProfile           bool
	injectLogLevel                string
	injectAPITokenSecret          string
	injectAppTokenSecret          string
	injectLogAsJSON               bool
	injectAppMaxConcurrency       int
	injectEnableMetrics           bool
	injectMetricsPort             int
	injectEnableDebug             bool
	injectEnv                     string
	injectCPULimit                string
	injectMemoryLimit             string
	injectCPURequest              string
	injectMemoryRequest           string
	injectListenAddresses         string
	injectLivenessProbeDelay      int
	injectLivenessProbeTimeout    int
	injectLivenessProbePeriod     int
	injectLivenessProbeThreshold  int
	injectReadinessProbeDelay     int
	injectReadinessProbeTimeout   int
	injectReadinessProbePeriod    int
	injectReadinessProbeThreshold int
	injectDaprImage               string
	injectAppSSL                  bool
	injectMaxRequestBodySize      int
	injectHTTPStreamRequestBody   bool
	injectGracefulShutdownSeconds int
)

var InjectCmd = &cobra.Command{
	Use:   "inject [flags] CONFIG-FILE",
	Short: "Inject dapr annotations into a Kubernetes configuration. Supported platforms: Kubernetes",
	Example: `
# Inject the first deployment found in the input
kubectl get deploy -l app=node -o yaml | dapr inject - | kubectl apply -f -

# Inject multiple deployments by name in a chain
kubectl get deploy -o yaml | dapr inject -r nodeapp - | dapr inject -r pythonapp | kubectl apply -f -

# Inject deployment in a specific namespace from file or directory by name
dapr inject -r nodeapp -n namespace mydeployment.yml | kubectl apply -f -

# Inject deployment from url by name
dapr inject -r nodeapp --log-level debug https://raw.githubusercontent.com/dapr/quickstarts/master/hello-kubernetes/deploy/node.yaml | kubectl apply -f -

--------------------------------------------------------------------------------
WARNING: If an app id is not provided, we will generate one using the format '<namespace>-<kind>-<name>'.
--------------------------------------------------------------------------------
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			print.FailureStatusEvent(os.Stderr, "please specify a kubernetes resource file")
			os.Exit(1)
		}

		input, err := readInput(args[0])
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}

		var config kubernetes.K8sInjectorConfig
		if injectTargetResource != "" {
			config = kubernetes.K8sInjectorConfig{
				TargetResource: &injectTargetResource,
			}  // nolint:exhaustivestruct
			if injectTargetNamespace != "" {
				config.TargetNamespace = &injectTargetNamespace
			}
		} else {
			if injectTargetNamespace != "" {
				// The resource is empty but namespace is set, this
				// is invalid as we cannot search for a resource
				// if the identifier isn't provided.
				print.FailureStatusEvent(os.Stderr, "--resource is required when --namespace is provided.")
				os.Exit(1)
			}
		}
		injector := kubernetes.NewK8sInjector(config)
		opts := getOptionsFromFlags()
		if err := injector.Inject(input, os.Stdout, opts); err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}
	},
}

func readInput(arg string) ([]io.Reader, error) {
	var inputs []io.Reader
	var err error
	if arg == "-" {
		// input is from stdin
		inputs = append(inputs, os.Stdin)
	} else if isURL(arg) {
		inputs, err = readInputsFromURL(arg)
		if err != nil {
			return nil, err
		}
	} else {
		// input is from file or dir
		inputs, err = readInputsFromFS(arg)
		if err != nil {
			return nil, err
		}
	}

	return inputs, nil
}

func readInputsFromURL(url string) ([]io.Reader, error) {
	resp, err := http.Get(url) // #nosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unable to read from %s: %d - %s", url, resp.StatusCode, resp.Status)
	}

	var b []byte
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(b)
	return []io.Reader{reader}, nil
}

func isURL(maybeURL string) bool {
	url, err := url.ParseRequestURI(maybeURL)
	if err != nil {
		return false
	}

	return url.Host != "" && url.Scheme != ""
}

func readInputsFromFS(path string) ([]io.Reader, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		// input is a file
		var file *os.File
		file, err = os.Open(path)
		if err != nil {
			return nil, err
		}

		return []io.Reader{file}, nil
	}

	// input is a directory
	var inputs []io.Reader
	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		inputs = append(inputs, file)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return inputs, nil
}

func getOptionsFromFlags() kubernetes.InjectOptions {
	// TODO: Use a pointer for int flag where zero is nil not -1
	o := []kubernetes.InjectOption{}
	if injectAppID != "" {
		o = append(o, kubernetes.WithAppID(injectAppID))
	}
	if injectConfig != "" {
		o = append(o, kubernetes.WithConfig(injectConfig))
	}
	if injectAppPort != -1 {
		o = append(o, kubernetes.WithAppPort(injectAppPort))
	}
	if injectAppProtocol != "" {
		o = append(o, kubernetes.WithAppProtocol(injectAppProtocol))
	}
	if injectEnableProfile {
		o = append(o, kubernetes.WithProfileEnabled())
	}
	if injectLogLevel != "" {
		o = append(o, kubernetes.WithLogLevel(injectLogLevel))
	}
	if injectAPITokenSecret != "" {
		o = append(o, kubernetes.WithAPITokenSecret(injectAPITokenSecret))
	}
	if injectAppTokenSecret != "" {
		o = append(o, kubernetes.WithAppTokenSecret(injectAppTokenSecret))
	}
	if injectLogAsJSON {
		o = append(o, kubernetes.WithLogAsJSON())
	}
	if injectAppMaxConcurrency != -1 {
		o = append(o, kubernetes.WithAppMaxConcurrency(injectAppMaxConcurrency))
	}
	if injectEnableMetrics {
		o = append(o, kubernetes.WithMetricsEnabled())
	}
	if injectMetricsPort != -1 {
		o = append(o, kubernetes.WithMetricsPort(injectMetricsPort))
	}
	if injectEnableDebug {
		o = append(o, kubernetes.WithDebugEnabled())
	}
	if injectEnv != "" {
		o = append(o, kubernetes.WithEnv(injectEnv))
	}
	if injectCPULimit != "" {
		o = append(o, kubernetes.WithCPULimit(injectCPULimit))
	}
	if injectMemoryLimit != "" {
		o = append(o, kubernetes.WithMemoryLimit(injectMemoryLimit))
	}
	if injectCPURequest != "" {
		o = append(o, kubernetes.WithCPURequest(injectCPURequest))
	}
	if injectMemoryRequest != "" {
		o = append(o, kubernetes.WithMemoryRequest(injectMemoryRequest))
	}
	if injectListenAddresses != "" {
		o = append(o, kubernetes.WithListenAddresses(injectListenAddresses))
	}
	if injectLivenessProbeDelay != -1 {
		o = append(o, kubernetes.WithLivenessProbeDelay(injectLivenessProbeDelay))
	}
	if injectLivenessProbeTimeout != -1 {
		o = append(o, kubernetes.WithLivenessProbeTimeout(injectLivenessProbeTimeout))
	}
	if injectLivenessProbePeriod != -1 {
		o = append(o, kubernetes.WithLivenessProbePeriod(injectLivenessProbePeriod))
	}
	if injectLivenessProbeThreshold != -1 {
		o = append(o, kubernetes.WithLivenessProbeThreshold(injectLivenessProbeThreshold))
	}
	if injectReadinessProbeDelay != -1 {
		o = append(o, kubernetes.WithReadinessProbeDelay(injectReadinessProbeDelay))
	}
	if injectReadinessProbeTimeout != -1 {
		o = append(o, kubernetes.WithReadinessProbeTimeout(injectReadinessProbeTimeout))
	}
	if injectReadinessProbePeriod != -1 {
		o = append(o, kubernetes.WithReadinessProbePeriod(injectReadinessProbePeriod))
	}
	if injectReadinessProbeThreshold != -1 {
		o = append(o, kubernetes.WithReadinessProbeThreshold(injectReadinessProbeThreshold))
	}
	if injectDaprImage != "" {
		o = append(o, kubernetes.WithDaprImage(injectDaprImage))
	}
	if injectAppSSL {
		o = append(o, kubernetes.WithAppSSL())
	}
	if injectMaxRequestBodySize != -1 {
		o = append(o, kubernetes.WithMaxRequestBodySize(injectMaxRequestBodySize))
	}
	if injectHTTPStreamRequestBody {
		o = append(o, kubernetes.WithHTTPStreamRequestBody())
	}
	if injectGracefulShutdownSeconds != -1 {
		o = append(o, kubernetes.WithGracefulShutdownSeconds(injectGracefulShutdownSeconds))
	}
	return kubernetes.NewInjectorOptions(o...)
}

func init() {
	InjectCmd.Flags().StringVarP(&injectTargetResource, "resource", "r", "", "The resource to target for injection")
	InjectCmd.Flags().StringVarP(&injectTargetNamespace, "namespace", "n", "", "The namespace the resource target is in (can only be set if --resource is also set)")
	InjectCmd.Flags().StringVarP(&injectAppID, "app-id", "a", "", "The app id to inject")
	InjectCmd.Flags().IntVarP(&injectAppPort, "app-port", "p", -1, "The port to expose the app on")
	InjectCmd.Flags().StringVarP(&injectConfig, "config", "c", "", "The config file to inject")
	InjectCmd.Flags().StringVar(&injectAppProtocol, "app-protocol", "", "The protocol to use for the app")
	InjectCmd.Flags().BoolVar(&injectEnableProfile, "enable-profile", false, "Enable profiling")
	InjectCmd.Flags().StringVar(&injectLogLevel, "log-level", "", "The log level to use")
	InjectCmd.Flags().StringVar(&injectAPITokenSecret, "api-token-secret", "", "The secret to use for the API token")
	InjectCmd.Flags().StringVar(&injectAppTokenSecret, "app-token-secret", "", "The secret to use for the app token")
	InjectCmd.Flags().BoolVar(&injectLogAsJSON, "log-as-json", false, "Log as JSON")
	InjectCmd.Flags().IntVar(&injectAppMaxConcurrency, "app-max-concurrency", -1, "The maximum number of concurrent requests to allow")
	InjectCmd.Flags().BoolVar(&injectEnableMetrics, "enable-metrics", false, "Enable metrics")
	InjectCmd.Flags().IntVar(&injectMetricsPort, "metrics-port", -1, "The port to expose the metrics on")
	InjectCmd.Flags().BoolVar(&injectEnableDebug, "enable-debug", false, "Enable debug")
	InjectCmd.Flags().StringVar(&injectEnv, "env", "", "Environment variables to set (key value pairs, comma separated)")
	InjectCmd.Flags().StringVar(&injectCPULimit, "cpu-limit", "", "The CPU limit to set")
	InjectCmd.Flags().StringVar(&injectMemoryLimit, "memory-limit", "", "The memory limit to set")
	InjectCmd.Flags().StringVar(&injectCPURequest, "cpu-request", "", "The CPU request to set")
	InjectCmd.Flags().StringVar(&injectMemoryRequest, "memory-request", "", "The memory request to set")
	InjectCmd.Flags().StringVar(&injectListenAddresses, "listen-addresses", "", "The addresses to listen on")
	InjectCmd.Flags().IntVar(&injectLivenessProbeDelay, "liveness-probe-delay", -1, "The delay to use for the liveness probe")
	InjectCmd.Flags().IntVar(&injectLivenessProbeTimeout, "liveness-probe-timeout", -1, "The timeout to use for the liveness probe")
	InjectCmd.Flags().IntVar(&injectLivenessProbePeriod, "liveness-probe-period", -1, "The period to use for the liveness probe")
	InjectCmd.Flags().IntVar(&injectLivenessProbeThreshold, "liveness-probe-threshold", -1, "The threshold to use for the liveness probe")
	InjectCmd.Flags().IntVar(&injectReadinessProbeDelay, "readiness-probe-delay", -1, "The delay to use for the readiness probe")
	InjectCmd.Flags().IntVar(&injectReadinessProbeTimeout, "readiness-probe-timeout", -1, "The timeout to use for the readiness probe")
	InjectCmd.Flags().IntVar(&injectReadinessProbePeriod, "readiness-probe-period", -1, "The period to use for the readiness probe")
	InjectCmd.Flags().IntVar(&injectReadinessProbeThreshold, "readiness-probe-threshold", -1, "The threshold to use for the readiness probe")
	InjectCmd.Flags().StringVar(&injectDaprImage, "dapr-image", "", "The image to use for the dapr sidecar container")
	InjectCmd.Flags().BoolVar(&injectAppSSL, "app-ssl", false, "Enable SSL for the app")
	InjectCmd.Flags().IntVar(&injectMaxRequestBodySize, "max-request-body-size", -1, "The maximum request body size to use")
	InjectCmd.Flags().BoolVar(&injectHTTPStreamRequestBody, "http-stream-request-body", false, "Enable streaming request body for HTTP")
	InjectCmd.Flags().IntVar(&injectGracefulShutdownSeconds, "graceful-shutdown-seconds", -1, "The number of seconds to wait for the app to shutdown")
	RootCmd.AddCommand(InjectCmd)
}
