package kubernetes

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type injection struct {
	targetName    string
	optionFactory func() InjectOptions
}

func TestInject(t *testing.T) {
	configs := []struct {
		testID           string
		injections       []injection
		inputFilePath    string
		expectedFilePath string
	}{
		{
			testID: "single targetted injection into pod config 1",
			injections: []injection{
				{
					targetName: "mypod",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("test-app"),
						)
					},
				},
			},
			inputFilePath:    "testdata/pod_raw.yml",
			expectedFilePath: "testdata/pod_injected_conf_1.yml",
		},
		{
			testID: "single targetted injection into pod config 2",
			injections: []injection{
				{
					targetName: "mypod",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("test-app"),
							WithProfileEnabled(),
							WithLogLevel("info"),
							WithDaprImage("custom-image"),
						)
					},
				},
			},
			inputFilePath:    "testdata/pod_raw.yml",
			expectedFilePath: "testdata/pod_injected_conf_2.yml",
		},
		{
			testID: "single targetted injection into deployment config 1",
			injections: []injection{
				{
					targetName: "nodeapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("nodeapp"),
							WithAppPort(3000),
						)
					},
				},
			},
			inputFilePath:    "testdata/deployment_raw.yml",
			expectedFilePath: "testdata/deployment_injected_conf_1.yml",
		},
		{
			testID: "partial injection into deployment config 1",
			injections: []injection{
				{
					targetName: "nodeapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("nodeapp"),
							WithAppPort(3000),
						)
					},
				},
			},
			inputFilePath:    "testdata/deployment_partial.yml",
			expectedFilePath: "testdata/deployment_injected_conf_1.yml",
		},
		{
			testID: "single targetted injection into multi config 1",
			injections: []injection{
				{
					targetName: "divideapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("divideapp"),
							WithAppPort(4000),
							WithConfig("appconfig"),
						)
					},
				},
			},
			inputFilePath:    "testdata/multi_raw.yml",
			expectedFilePath: "testdata/multi_injected_conf_1.yml",
		},
		{
			testID: "multiple targetted injections into multi config 2",
			injections: []injection{
				{
					targetName: "subtractapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("subtractapp"),
							WithAppPort(80),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetName: "addapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("addapp"),
							WithAppPort(6000),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetName: "multiplyapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("multiplyapp"),
							WithAppPort(5000),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetName: "divideapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("divideapp"),
							WithAppPort(4000),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetName: "calculator-front-end",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("calculator-front-end"),
							WithAppPort(8080),
							WithConfig("appconfig"),
						)
					},
				},
			},
			inputFilePath:    "testdata/multi_raw.yml",
			expectedFilePath: "testdata/multi_injected_conf_2.yml",
		},
		{
			testID: "single untargetted injection into multi config 3",
			injections: []injection{
				{
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("subtractapp"),
							WithAppPort(80),
							WithConfig("appconfig"),
						)
					},
				},
			},
			inputFilePath:    "testdata/multi_raw.yml",
			expectedFilePath: "testdata/multi_injected_conf_3.yml",
		},
		{
			testID: "single untargetted injection into list config",
			injections: []injection{
				{
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppId("nodeapp"),
							WithAppPort(3000),
						)
					},
				},
			},
			inputFilePath:    "testdata/list_raw.yml",
			expectedFilePath: "testdata/list_injected_conf_1.yml",
		},
	}

	for _, tt := range configs {
		t.Run(tt.testID, func(t *testing.T) {
			inputFile, err := os.Open(tt.inputFilePath)
			assert.NoError(t, err)

			defer func() {
				err := inputFile.Close()
				assert.NoError(t, err)
			}()

			// Iterate through all the injections and pipe them together.
			var out bytes.Buffer
			in := []io.Reader{inputFile}
			for i, injection := range tt.injections {
				injector := NewK8sInjector(K8sInjectorConfig{
					TargetResource: &injection.targetName,
				})
				injectOptions := injection.optionFactory()

				out.Reset()
				err = injector.Inject(in, &out, injectOptions)
				assert.NoError(t, err)

				// if it isn't the last injection then set input to this injection output.
				if i != len(tt.injections)-1 {
					outReader := strings.NewReader(out.String())
					in = []io.Reader{outReader}
				}
			}

			// Split the multi-document string into individual documents for comparison.
			outString := out.String()
			outDocs := strings.Split(outString, "---")

			expected, err := ioutil.ReadFile(tt.expectedFilePath)
			assert.NoError(t, err)

			expectedString := string(expected)
			expectedDocs := strings.Split(expectedString, "---")

			// We must sort the documents to ensure we are comparing the correct documents.
			// The content of the documents should be equivalent but it will not be the same
			// as the order of keys are not being preserved. Therefore, we sort on the content
			// length instead. This isn't perfect as additional character may be included but
			// as long as we have enough spread between the documents we should be ok to use this
			// to get an order. sortDocs will panic if it tries to compare content that is the
			// same length as we would lose ordering.
			sortDocs(outDocs)
			sortDocs(expectedDocs)
			assert.Equal(t, len(expectedDocs), len(outDocs))

			for i := range expectedDocs {
				assert.YAMLEq(t, expectedDocs[i], outDocs[i])
			}
		})
	}
}

func sortDocs(docs []string) {
	sort.Slice(docs, func(i, j int) bool {
		if len(docs[i]) == len(docs[j]) {
			panic("Cannot sort docs with the same length, please ensure tests docs are a unique length.")
		}
		return len(docs[i]) < len(docs[j])
	})
}

func TestGetDaprAnnotations(t *testing.T) {
	t.Run("get dapr annotations", func(t *testing.T) {
		config := NewInjectorOptions(
			WithAppId("test-app"),
			WithMetricsEnabled(),
			WithMetricsPort(9090),
			WithAPITokenSecret("test-api-token"),
			WithAppTokenSecret("test-app-token"),
			WithAppMaxConcurrency(2),
			WithAppPort(8080),
			WithAppProtocol("http"),
			WithAppSSL(),
			WithCPULimit("0.5"),
			WithMemoryLimit("512Mi"),
			WithCPURequest("0.1"),
			WithMemoryRequest("256Mi"),
			WithConfig("test-config"),
			WithDebugEnabled(),
			WithEnv("key=value,key1=value1"),
			WithLogAsJson(),
			WithListenAddresses("0.0.0.0"),
			WithDaprImage("test-image"),
			WithProfileEnabled(),
			WithMaxRequestBodySize(8),
			WithReadBufferSize(4),
			WithReadinessProbeDelay(10),
			WithReadinessProbePeriod(30),
			WithReadinessProbeThreshold(15),
			WithReadinessProbeTimeout(10),
			WithLivenessProbeDelay(10),
			WithLivenessProbePeriod(30),
			WithLivenessProbeThreshold(15),
			WithLivenessProbeTimeout(10),
			WithLogLevel("debug"),
			WithGracefulShutdownSeconds(10),
		)

		annotations := getDaprAnnotations(&config)

		assert.Equal(t, annotations["dapr.io/enabled"], "true")
		assert.Equal(t, annotations["dapr.io/app-id"], "test-app")
		assert.Equal(t, annotations["dapr.io/app-port"], "8080")
		assert.Equal(t, annotations["dapr.io/config"], "test-config")
		assert.Equal(t, annotations["dapr.io/app-protocol"], "http")
		assert.Equal(t, annotations["dapr.io/enable-profiling"], "true")
		assert.Equal(t, annotations["dapr.io/log-level"], "debug")
		assert.Equal(t, annotations["dapr.io/api-token-secret"], "test-api-token")
		assert.Equal(t, annotations["dapr.io/app-token-secret"], "test-app-token")
		assert.Equal(t, annotations["dapr.io/log-as-json"], "true")
		assert.Equal(t, annotations["dapr.io/app-max-concurrency"], "8")
		assert.Equal(t, annotations["dapr.io/enable-metrics"], "true")
		assert.Equal(t, annotations["dapr.io/metrics-port"], "9090")
		assert.Equal(t, annotations["dapr.io/enable-debug"], "true")
		assert.Equal(t, annotations["dapr.io/env"], "key=value,key1=value1")
		assert.Equal(t, annotations["dapr.io/sidecar-cpu-limit"], "0.5")
		assert.Equal(t, annotations["dapr.io/sidecar-memory-limit"], "512Mi")
		assert.Equal(t, annotations["dapr.io/sidecar-cpu-request"], "0.1")
		assert.Equal(t, annotations["dapr.io/sidecar-memory-request"], "256Mi")
		assert.Equal(t, annotations["dapr.io/sidecar-listen-addresses"], "0.0.0.0")
		assert.Equal(t, annotations["dapr.io/sidecar-liveness-probe-delay-seconds"], "10")
		assert.Equal(t, annotations["dapr.io/sidecar-liveness-probe-timeout-seconds"], "10")
		assert.Equal(t, annotations["dapr.io/sidecar-liveness-probe-period-seconds"], "30")
		assert.Equal(t, annotations["dapr.io/sidecar-liveness-probe-threshold"], "15")
		assert.Equal(t, annotations["dapr.io/sidecar-readiness-probe-delay-seconds"], "10")
		assert.Equal(t, annotations["dapr.io/sidecar-readiness-probe-timeout-seconds"], "10")
		assert.Equal(t, annotations["dapr.io/sidecar-readiness-probe-period-seconds"], "30")
		assert.Equal(t, annotations["dapr.io/sidecar-readiness-probe-threshold"], "15")
		assert.Equal(t, annotations["dapr.io/sidecar-image"], "test-image")
		assert.Equal(t, annotations["dapr.io/app-ssl"], "true")
		assert.Equal(t, annotations["dapr.io/http-max-request-size"], "8")
		assert.Equal(t, annotations["dapr.io/http-read-buffer-size"], "4")
		assert.Equal(t, annotations["dapr.io/http-stream-request-body"], "true")
		assert.Equal(t, annotations["dapr.io/graceful-shutdown-seconds"], "10")
	})
}
