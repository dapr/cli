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
	// Helper function used to order test documents.
	sortDocs := func(docs []string) {
		sort.Slice(docs, func(i, j int) bool {
			if len(docs[i]) == len(docs[j]) {
				panic("Cannot sort docs with the same length, please ensure tests docs are a unique length.")
			}
			return len(docs[i]) < len(docs[j])
		})
	}

	configs := []struct {
		testID           string
		injections       []injection
		inputFilePath    string
		expectedFilePath string
	}{
		{
			testID: "single targeted injection into pod config 1",
			injections: []injection{
				{
					targetName: "mypod",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("test-app"),
						)
					},
				},
			},
			inputFilePath:    "testdata/pod_raw.yml",
			expectedFilePath: "testdata/pod_injected_conf_1.yml",
		},
		{
			testID: "single targeted injection into pod config 2",
			injections: []injection{
				{
					targetName: "mypod",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("test-app"),
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
			testID: "single targeted injection into deployment config 1",
			injections: []injection{
				{
					targetName: "nodeapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("nodeapp"),
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
							WithAppID("nodeapp"),
							WithAppPort(3000),
						)
					},
				},
			},
			inputFilePath:    "testdata/deployment_partial.yml",
			expectedFilePath: "testdata/deployment_injected_conf_1.yml",
		},
		{
			testID: "single targeted injection into multi config 1",
			injections: []injection{
				{
					targetName: "divideapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("divideapp"),
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
			testID: "multiple targeted injections into multi config 2",
			injections: []injection{
				{
					targetName: "subtractapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("subtractapp"),
							WithAppPort(80),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetName: "addapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("addapp"),
							WithAppPort(6000),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetName: "multiplyapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("multiplyapp"),
							WithAppPort(5000),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetName: "divideapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("divideapp"),
							WithAppPort(4000),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetName: "calculator-front-end",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("calculator-front-end"),
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
			testID: "single untargeted injection into multi config 3",
			injections: []injection{
				{
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("subtractapp"),
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
			testID: "single untargeted injection into list config",
			injections: []injection{
				{
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("nodeapp"),
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
				err = inputFile.Close()
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

func TestGetDaprAnnotations(t *testing.T) {
	t.Run("get dapr annotations", func(t *testing.T) {
		config := NewInjectorOptions(
			WithAppID("test-app"),
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
			WithDebugPort(9050),
			WithEnv("key=value,key1=value1"),
			WithLogAsJSON(),
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
			WithReadBufferSize(4),
			WithLogLevel("debug"),
			WithGracefulShutdownSeconds(10),
		)

		annotations := getDaprAnnotations(&config)

		assert.Equal(t, "true", annotations[daprEnabledKey])
		assert.Equal(t, annotations[daprAppIDKey], "test-app")
		assert.Equal(t, annotations[daprAppPortKey], "8080")
		assert.Equal(t, annotations[daprConfigKey], "test-config")
		assert.Equal(t, annotations[daprAppProtocolKey], "http")
		assert.Equal(t, annotations[daprEnableProfilingKey], "true")
		assert.Equal(t, annotations[daprLogLevelKey], "debug")
		assert.Equal(t, annotations[daprAPITokenSecretKey], "test-api-token")
		assert.Equal(t, annotations[daprAppTokenSecretKey], "test-app-token")
		assert.Equal(t, annotations[daprLogAsJSONKey], "true")
		assert.Equal(t, annotations[daprAppMaxConcurrencyKey], "2")
		assert.Equal(t, annotations[daprEnableMetricsKey], "true")
		assert.Equal(t, annotations[daprMetricsPortKey], "9090")
		assert.Equal(t, annotations[daprEnableDebugKey], "true")
		assert.Equal(t, annotations[daprDebugPortKey], "9050")
		assert.Equal(t, annotations[daprEnvKey], "key=value,key1=value1")
		assert.Equal(t, annotations[daprCPULimitKey], "0.5")
		assert.Equal(t, annotations[daprMemoryLimitKey], "512Mi")
		assert.Equal(t, annotations[daprCPURequestKey], "0.1")
		assert.Equal(t, annotations[daprMemoryRequestKey], "256Mi")
		assert.Equal(t, annotations[daprListenAddressesKey], "0.0.0.0")
		assert.Equal(t, annotations[daprLivenessProbeDelayKey], "10")
		assert.Equal(t, annotations[daprLivenessProbeTimeoutKey], "10")
		assert.Equal(t, annotations[daprLivenessProbePeriodKey], "30")
		assert.Equal(t, annotations[daprLivenessProbeThresholdKey], "15")
		assert.Equal(t, annotations[daprReadinessProbeDelayKey], "10")
		assert.Equal(t, annotations[daprReadinessProbeTimeoutKey], "10")
		assert.Equal(t, annotations[daprReadinessProbePeriodKey], "30")
		assert.Equal(t, annotations[daprReadinessProbeThresholdKey], "15")
		assert.Equal(t, annotations[daprImageKey], "test-image")
		assert.Equal(t, annotations[daprAppSSLKey], "true")
		assert.Equal(t, annotations[daprMaxRequestBodySizeKey], "8")
		assert.Equal(t, annotations[daprReadBufferSizeKey], "4")
		assert.Equal(t, annotations[daprHTTPStreamRequestBodyKey], "true")
		assert.Equal(t, annotations[daprGracefulShutdownSecondsKey], "10")
	})
}
