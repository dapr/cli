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
	annotateTargetResource          string
	annotateTargetNamespace         string
	annotateAppID                   string
	annotateAppPort                 int
	annotateConfig                  string
	annotateAppProtocol             string
	annotateEnableProfile           bool
	annotateLogLevel                string
	annotateAPITokenSecret          string
	annotateAppTokenSecret          string
	annotateLogAsJSON               bool
	annotateAppMaxConcurrency       int
	annotateEnableMetrics           bool
	annotateMetricsPort             int
	annotateEnableDebug             bool
	annotateEnv                     string
	annotateCPULimit                string
	annotateMemoryLimit             string
	annotateCPURequest              string
	annotateMemoryRequest           string
	annotateListenAddresses         string
	annotateLivenessProbeDelay      int
	annotateLivenessProbeTimeout    int
	annotateLivenessProbePeriod     int
	annotateLivenessProbeThreshold  int
	annotateReadinessProbeDelay     int
	annotateReadinessProbeTimeout   int
	annotateReadinessProbePeriod    int
	annotateReadinessProbeThreshold int
	annotateDaprImage               string
	annotateAppSSL                  bool
	annotateMaxRequestBodySize      int
	annotateHTTPStreamRequestBody   bool
	annotateGracefulShutdownSeconds int
)

var AnnotateCmd = &cobra.Command{
	Use:   "annotate [flags] CONFIG-FILE",
	Short: "Add dapr annotations to a Kubernetes configuration. Supported platforms: Kubernetes",
	Example: `
# Annotate the first deployment found in the input
kubectl get deploy -l app=node -o yaml | dapr annotate - | kubectl apply -f -

# Annotate multiple deployments by name in a chain
kubectl get deploy -o yaml | dapr annotate -r nodeapp - | dapr annotate -r pythonapp - | kubectl apply -f -

# Annotate deployment in a specific namespace from file or directory by name
dapr annotate -r nodeapp -n namespace mydeploy.yaml | kubectl apply -f -

# Annotate deployment from url by name
dapr annotate -r nodeapp --log-level debug https://raw.githubusercontent.com/dapr/quickstarts/master/tutorials/hello-kubernetes/deploy/node.yaml | kubectl apply -f -

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

		var config kubernetes.K8sAnnotatorConfig
		if annotateTargetResource != "" {
			config = kubernetes.K8sAnnotatorConfig{
				TargetResource: &annotateTargetResource,
			} // nolint:exhaustivestruct
			if annotateTargetNamespace != "" {
				config.TargetNamespace = &annotateTargetNamespace
			}
		} else {
			if annotateTargetNamespace != "" {
				// The resource is empty but namespace is set, this
				// is invalid as we cannot search for a resource
				// if the identifier isn't provided.
				print.FailureStatusEvent(os.Stderr, "--resource is required when --namespace is provided.")
				os.Exit(1)
			}
		}
		annotator := kubernetes.NewK8sAnnotator(config)
		opts := getOptionsFromFlags()
		if err := annotator.Annotate(input, os.Stdout, opts); err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}
	},
}

func readInput(arg string) ([]io.Reader, error) {
	var inputs []io.Reader
	var err error
	if arg == "-" {
		// input is from stdin.
		inputs = append(inputs, os.Stdin)
	} else if isURL(arg) {
		inputs, err = readInputsFromURL(arg)
		if err != nil {
			return nil, err
		}
	} else {
		// input is from file or dir.
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
		// input is a file.
		var file *os.File
		file, err = os.Open(path)
		if err != nil {
			return nil, err
		}

		return []io.Reader{file}, nil
	}

	// input is a directory.
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

func getOptionsFromFlags() kubernetes.AnnotateOptions {
	// TODO: Use a pointer for int flag where zero is nil not -1.
	o := []kubernetes.AnnoteOption{}
	if annotateAppID != "" {
		o = append(o, kubernetes.WithAppID(annotateAppID))
	}
	if annotateConfig != "" {
		o = append(o, kubernetes.WithConfig(annotateConfig))
	}
	if annotateAppPort != -1 {
		o = append(o, kubernetes.WithAppPort(annotateAppPort))
	}
	if annotateAppProtocol != "" {
		o = append(o, kubernetes.WithAppProtocol(annotateAppProtocol))
	}
	if annotateEnableProfile {
		o = append(o, kubernetes.WithProfileEnabled())
	}
	if annotateLogLevel != "" {
		o = append(o, kubernetes.WithLogLevel(annotateLogLevel))
	}
	if annotateAPITokenSecret != "" {
		o = append(o, kubernetes.WithAPITokenSecret(annotateAPITokenSecret))
	}
	if annotateAppTokenSecret != "" {
		o = append(o, kubernetes.WithAppTokenSecret(annotateAppTokenSecret))
	}
	if annotateLogAsJSON {
		o = append(o, kubernetes.WithLogAsJSON())
	}
	if annotateAppMaxConcurrency != -1 {
		o = append(o, kubernetes.WithAppMaxConcurrency(annotateAppMaxConcurrency))
	}
	if annotateEnableMetrics {
		o = append(o, kubernetes.WithMetricsEnabled())
	}
	if annotateMetricsPort != -1 {
		o = append(o, kubernetes.WithMetricsPort(annotateMetricsPort))
	}
	if annotateEnableDebug {
		o = append(o, kubernetes.WithDebugEnabled())
	}
	if annotateEnv != "" {
		o = append(o, kubernetes.WithEnv(annotateEnv))
	}
	if annotateCPULimit != "" {
		o = append(o, kubernetes.WithCPULimit(annotateCPULimit))
	}
	if annotateMemoryLimit != "" {
		o = append(o, kubernetes.WithMemoryLimit(annotateMemoryLimit))
	}
	if annotateCPURequest != "" {
		o = append(o, kubernetes.WithCPURequest(annotateCPURequest))
	}
	if annotateMemoryRequest != "" {
		o = append(o, kubernetes.WithMemoryRequest(annotateMemoryRequest))
	}
	if annotateListenAddresses != "" {
		o = append(o, kubernetes.WithListenAddresses(annotateListenAddresses))
	}
	if annotateLivenessProbeDelay != -1 {
		o = append(o, kubernetes.WithLivenessProbeDelay(annotateLivenessProbeDelay))
	}
	if annotateLivenessProbeTimeout != -1 {
		o = append(o, kubernetes.WithLivenessProbeTimeout(annotateLivenessProbeTimeout))
	}
	if annotateLivenessProbePeriod != -1 {
		o = append(o, kubernetes.WithLivenessProbePeriod(annotateLivenessProbePeriod))
	}
	if annotateLivenessProbeThreshold != -1 {
		o = append(o, kubernetes.WithLivenessProbeThreshold(annotateLivenessProbeThreshold))
	}
	if annotateReadinessProbeDelay != -1 {
		o = append(o, kubernetes.WithReadinessProbeDelay(annotateReadinessProbeDelay))
	}
	if annotateReadinessProbeTimeout != -1 {
		o = append(o, kubernetes.WithReadinessProbeTimeout(annotateReadinessProbeTimeout))
	}
	if annotateReadinessProbePeriod != -1 {
		o = append(o, kubernetes.WithReadinessProbePeriod(annotateReadinessProbePeriod))
	}
	if annotateReadinessProbeThreshold != -1 {
		o = append(o, kubernetes.WithReadinessProbeThreshold(annotateReadinessProbeThreshold))
	}
	if annotateDaprImage != "" {
		o = append(o, kubernetes.WithDaprImage(annotateDaprImage))
	}
	if annotateAppSSL {
		o = append(o, kubernetes.WithAppSSL())
	}
	if annotateMaxRequestBodySize != -1 {
		o = append(o, kubernetes.WithMaxRequestBodySize(annotateMaxRequestBodySize))
	}
	if annotateHTTPStreamRequestBody {
		o = append(o, kubernetes.WithHTTPStreamRequestBody())
	}
	if annotateGracefulShutdownSeconds != -1 {
		o = append(o, kubernetes.WithGracefulShutdownSeconds(annotateGracefulShutdownSeconds))
	}
	return kubernetes.NewAnnotateOptions(o...)
}

func init() {
	AnnotateCmd.Flags().StringVarP(&annotateTargetResource, "resource", "r", "", "The resource to target to annotate")
	AnnotateCmd.Flags().StringVarP(&annotateTargetNamespace, "namespace", "n", "", "The namespace the resource target is in (can only be set if --resource is also set)")
	AnnotateCmd.Flags().StringVarP(&annotateAppID, "app-id", "a", "", "The app id to annotate")
	AnnotateCmd.Flags().IntVarP(&annotateAppPort, "app-port", "p", -1, "The port to expose the app on")
	AnnotateCmd.Flags().StringVarP(&annotateConfig, "config", "c", "", "The config file to annotate")
	AnnotateCmd.Flags().StringVar(&annotateAppProtocol, "app-protocol", "", "The protocol to use for the app")
	AnnotateCmd.Flags().BoolVar(&annotateEnableProfile, "enable-profile", false, "Enable profiling")
	AnnotateCmd.Flags().StringVar(&annotateLogLevel, "log-level", "", "The log level to use")
	AnnotateCmd.Flags().StringVar(&annotateAPITokenSecret, "api-token-secret", "", "The secret to use for the API token")
	AnnotateCmd.Flags().StringVar(&annotateAppTokenSecret, "app-token-secret", "", "The secret to use for the app token")
	AnnotateCmd.Flags().BoolVar(&annotateLogAsJSON, "log-as-json", false, "Log as JSON")
	AnnotateCmd.Flags().IntVar(&annotateAppMaxConcurrency, "app-max-concurrency", -1, "The maximum number of concurrent requests to allow")
	AnnotateCmd.Flags().BoolVar(&annotateEnableMetrics, "enable-metrics", false, "Enable metrics")
	AnnotateCmd.Flags().IntVar(&annotateMetricsPort, "metrics-port", -1, "The port to expose the metrics on")
	AnnotateCmd.Flags().BoolVar(&annotateEnableDebug, "enable-debug", false, "Enable debug")
	AnnotateCmd.Flags().StringVar(&annotateEnv, "env", "", "Environment variables to set (key value pairs, comma separated)")
	AnnotateCmd.Flags().StringVar(&annotateCPULimit, "cpu-limit", "", "The CPU limit to set")
	AnnotateCmd.Flags().StringVar(&annotateMemoryLimit, "memory-limit", "", "The memory limit to set")
	AnnotateCmd.Flags().StringVar(&annotateCPURequest, "cpu-request", "", "The CPU request to set")
	AnnotateCmd.Flags().StringVar(&annotateMemoryRequest, "memory-request", "", "The memory request to set")
	AnnotateCmd.Flags().StringVar(&annotateListenAddresses, "listen-addresses", "", "The addresses to listen on")
	AnnotateCmd.Flags().IntVar(&annotateLivenessProbeDelay, "liveness-probe-delay", -1, "The delay to use for the liveness probe")
	AnnotateCmd.Flags().IntVar(&annotateLivenessProbeTimeout, "liveness-probe-timeout", -1, "The timeout to use for the liveness probe")
	AnnotateCmd.Flags().IntVar(&annotateLivenessProbePeriod, "liveness-probe-period", -1, "The period to use for the liveness probe")
	AnnotateCmd.Flags().IntVar(&annotateLivenessProbeThreshold, "liveness-probe-threshold", -1, "The threshold to use for the liveness probe")
	AnnotateCmd.Flags().IntVar(&annotateReadinessProbeDelay, "readiness-probe-delay", -1, "The delay to use for the readiness probe")
	AnnotateCmd.Flags().IntVar(&annotateReadinessProbeTimeout, "readiness-probe-timeout", -1, "The timeout to use for the readiness probe")
	AnnotateCmd.Flags().IntVar(&annotateReadinessProbePeriod, "readiness-probe-period", -1, "The period to use for the readiness probe")
	AnnotateCmd.Flags().IntVar(&annotateReadinessProbeThreshold, "readiness-probe-threshold", -1, "The threshold to use for the readiness probe")
	AnnotateCmd.Flags().StringVar(&annotateDaprImage, "dapr-image", "", "The image to use for the dapr sidecar container")
	AnnotateCmd.Flags().BoolVar(&annotateAppSSL, "app-ssl", false, "Enable SSL for the app")
	AnnotateCmd.Flags().IntVar(&annotateMaxRequestBodySize, "max-request-body-size", -1, "The maximum request body size to use")
	AnnotateCmd.Flags().BoolVar(&annotateHTTPStreamRequestBody, "http-stream-request-body", false, "Enable streaming request body for HTTP")
	AnnotateCmd.Flags().IntVar(&annotateGracefulShutdownSeconds, "graceful-shutdown-seconds", -1, "The number of seconds to wait for the app to shutdown")
	RootCmd.AddCommand(AnnotateCmd)
}
