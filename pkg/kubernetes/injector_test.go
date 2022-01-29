package kubernetes

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	injectorTestDataDir = "testdata/injector"
)

var (
	podInDir         = path.Join(injectorTestDataDir, "pod/in")
	podOutDir        = path.Join(injectorTestDataDir, "pod/out")
	deploymentInDir  = path.Join(injectorTestDataDir, "deployment/in")
	deploymentOutDir = path.Join(injectorTestDataDir, "deployment/out")
	multiInDir       = path.Join(injectorTestDataDir, "multi/in")
	multiOutDir      = path.Join(injectorTestDataDir, "multi/out")
	listInDir        = path.Join(injectorTestDataDir, "list/in")
	listOutDir       = path.Join(injectorTestDataDir, "list/out")
)

type injection struct {
	targetResource  string
	targetNamespace string
	optionFactory   func() InjectOptions
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
		printOutput      bool
	}{
		{
			testID: "single targeted injection into pod (config 1)",
			injections: []injection{
				{
					targetResource: "mypod",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("test-app"),
						)
					},
				},
			},
			inputFilePath:    path.Join(podInDir, "raw.yml"),
			expectedFilePath: path.Join(podOutDir, "config_1.yml"),
			// printOutput:      true, // Uncomment to debug.
		},
		{
			testID: "single targeted injection into pod (config 2)",
			injections: []injection{
				{
					targetResource: "mypod",
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
			inputFilePath:    path.Join(podInDir, "raw.yml"),
			expectedFilePath: path.Join(podOutDir, "config_2.yml"),
			// printOutput:      true, // Uncomment to debug.
		},
		{
			testID: "single targeted injection into pod without an app id in default namespace (config 3)",
			injections: []injection{
				{
					targetResource: "mypod",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions()
					},
				},
			},
			inputFilePath:    path.Join(podInDir, "raw.yml"),
			expectedFilePath: path.Join(podOutDir, "config_3.yml"),
			// printOutput:      true, // Uncomment to debug.
		},
		{
			testID: "single targeted injection into pod without an app id in a namespace (config 4)",
			injections: []injection{
				{
					targetResource: "mypod",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions()
					},
				},
			},
			inputFilePath:    path.Join(podInDir, "namespace.yml"),
			expectedFilePath: path.Join(podOutDir, "config_4.yml"),
			// printOutput:      true, // Uncomment to debug.
		},
		{
			testID: "single targeted injection into deployment (config 1)",
			injections: []injection{
				{
					targetResource: "nodeapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("nodeapp"),
							WithAppPort(3000),
						)
					},
				},
			},
			inputFilePath:    path.Join(deploymentInDir, "raw.yml"),
			expectedFilePath: path.Join(deploymentOutDir, "config_1.yml"),
			// printOutput:      true, // Uncomment to debug.
		},
		{
			testID: "partial injection into deployment (config 2)",
			injections: []injection{
				{
					targetResource: "nodeapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("nodeapp"),
							WithAppPort(3000),
						)
					},
				},
			},
			inputFilePath:    path.Join(deploymentInDir, "partial.yml"),
			expectedFilePath: path.Join(deploymentOutDir, "config_1.yml"),
			// printOutput:      true, // Uncomment to debug.
		},
		{
			testID: "single targeted injection into multi (config 1)",
			injections: []injection{
				{
					targetResource: "divideapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("divideapp"),
							WithAppPort(4000),
							WithConfig("appconfig"),
						)
					},
				},
			},
			inputFilePath:    path.Join(multiInDir, "raw.yml"),
			expectedFilePath: path.Join(multiOutDir, "config_1.yml"),
			// printOutput:      true, // Uncomment to debug.
		},
		{
			testID: "multiple targeted injections into multi (config 2)",
			injections: []injection{
				{
					targetResource: "subtractapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("subtractapp"),
							WithAppPort(80),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetResource: "addapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("addapp"),
							WithAppPort(6000),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetResource: "multiplyapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("multiplyapp"),
							WithAppPort(5000),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetResource: "divideapp",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("divideapp"),
							WithAppPort(4000),
							WithConfig("appconfig"),
						)
					},
				},
				{
					targetResource: "calculator-front-end",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("calculator-front-end"),
							WithAppPort(8080),
							WithConfig("appconfig"),
						)
					},
				},
			},
			inputFilePath:    path.Join(multiInDir, "raw.yml"),
			expectedFilePath: path.Join(multiOutDir, "config_2.yml"),
			// printOutput:      true, // Uncomment to debug.
		},
		{
			testID: "single untargeted injection into multi (config 3)",
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
			inputFilePath:    path.Join(multiInDir, "raw.yml"),
			expectedFilePath: path.Join(multiOutDir, "config_3.yml"),
			// printOutput:      true, // Uncomment to debug.
		},
		{
			testID: "single targeted injection into multi with namespace (config 4)",
			injections: []injection{
				{
					targetResource:  "subtractapp",
					targetNamespace: "test1",
					optionFactory: func() InjectOptions {
						return NewInjectorOptions(
							WithAppID("subtractapp"),
						)
					},
				},
			},
			inputFilePath:    path.Join(multiInDir, "namespace.yml"),
			expectedFilePath: path.Join(multiOutDir, "config_4.yml"),
			// printOutput:      true, // Uncomment to debug.
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
			inputFilePath:    path.Join(listInDir, "raw.yml"),
			expectedFilePath: path.Join(listOutDir, "config_1.yml"),
			// printOutput:      true, // Uncomment to debug.
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
					TargetResource:  &injection.targetResource,
					TargetNamespace: &injection.targetNamespace,
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
				if tt.printOutput {
					t.Logf(outDocs[i])
				}
				assert.YAMLEq(t, expectedDocs[i], outDocs[i])
			}
		})
	}
}

func TestGetDaprAnnotations(t *testing.T) {
	t.Run("get dapr annotations", func(t *testing.T) {
		appID := "test-app"
		metricsPort := 9090
		apiTokenSecret := "test-api-token-secret" // #nosec
		appTokenSecret := "test-app-token-secret" // #nosec
		appMaxConcurrency := 2
		appPort := 8080
		appProtocol := "http"
		cpuLimit := "0.5"
		memoryLimit := "512Mi"
		cpuRequest := "0.1"
		memoryRequest := "256Mi"
		config := "appconfig"
		debugPort := 9091
		env := "key=value key1=value1"
		listenAddresses := "0.0.0.0"
		daprImage := "test-iamge"
		maxRequestBodySize := 8
		readBufferSize := 4
		livenessProbeDelay := 10
		livenessProbePeriod := 20
		livenessProbeThreshold := 3
		livenessProbeTimeout := 30
		readinessProbeDelay := 40
		readinessProbePeriod := 50
		readinessProbeThreshold := 6
		readinessProbeTimeout := 60
		logLevel := "debug"
		gracefulShutdownSeconds := 10

		opts := NewInjectorOptions(
			WithAppID(appID),
			WithMetricsEnabled(),
			WithMetricsPort(metricsPort),
			WithAPITokenSecret(apiTokenSecret),
			WithAppTokenSecret(appTokenSecret),
			WithAppMaxConcurrency(appMaxConcurrency),
			WithAppPort(appPort),
			WithAppProtocol(appProtocol),
			WithAppSSL(),
			WithCPULimit(cpuLimit),
			WithMemoryLimit(memoryLimit),
			WithCPURequest(cpuRequest),
			WithMemoryRequest(memoryRequest),
			WithConfig(config),
			WithDebugEnabled(),
			WithDebugPort(debugPort),
			WithEnv(env),
			WithLogAsJSON(),
			WithListenAddresses(listenAddresses),
			WithDaprImage(daprImage),
			WithProfileEnabled(),
			WithMaxRequestBodySize(maxRequestBodySize),
			WithReadBufferSize(readBufferSize),
			WithReadinessProbeDelay(readinessProbeDelay),
			WithReadinessProbePeriod(readinessProbePeriod),
			WithReadinessProbeThreshold(readinessProbeThreshold),
			WithReadinessProbeTimeout(readinessProbeTimeout),
			WithLivenessProbeDelay(livenessProbeDelay),
			WithLivenessProbePeriod(livenessProbePeriod),
			WithLivenessProbeThreshold(livenessProbeThreshold),
			WithLivenessProbeTimeout(livenessProbeTimeout),
			WithLogLevel(logLevel),
			WithHTTPStreamRequestBody(),
			WithGracefulShutdownSeconds(gracefulShutdownSeconds),
		)

		annotations := getDaprAnnotations(&opts)

		assert.Equal(t, "true", annotations[daprEnabledKey])
		assert.Equal(t, appID, annotations[daprAppIDKey])
		assert.Equal(t, fmt.Sprintf("%d", appPort), annotations[daprAppPortKey])
		assert.Equal(t, config, annotations[daprConfigKey])
		assert.Equal(t, appProtocol, annotations[daprAppProtocolKey])
		assert.Equal(t, "true", annotations[daprEnableProfilingKey])
		assert.Equal(t, logLevel, annotations[daprLogLevelKey])
		assert.Equal(t, apiTokenSecret, annotations[daprAPITokenSecretKey])
		assert.Equal(t, appTokenSecret, annotations[daprAppTokenSecretKey])
		assert.Equal(t, "true", annotations[daprLogAsJSONKey])
		assert.Equal(t, fmt.Sprintf("%d", appMaxConcurrency), annotations[daprAppMaxConcurrencyKey])
		assert.Equal(t, "true", annotations[daprEnableMetricsKey])
		assert.Equal(t, fmt.Sprintf("%d", metricsPort), annotations[daprMetricsPortKey])
		assert.Equal(t, "true", annotations[daprEnableDebugKey])
		assert.Equal(t, fmt.Sprintf("%d", debugPort), annotations[daprDebugPortKey])
		assert.Equal(t, env, annotations[daprEnvKey])
		assert.Equal(t, cpuLimit, annotations[daprCPULimitKey])
		assert.Equal(t, memoryLimit, annotations[daprMemoryLimitKey])
		assert.Equal(t, cpuRequest, annotations[daprCPURequestKey])
		assert.Equal(t, memoryRequest, annotations[daprMemoryRequestKey])
		assert.Equal(t, listenAddresses, annotations[daprListenAddressesKey])
		assert.Equal(t, fmt.Sprintf("%d", livenessProbeDelay), annotations[daprLivenessProbeDelayKey])
		assert.Equal(t, fmt.Sprintf("%d", livenessProbeTimeout), annotations[daprLivenessProbeTimeoutKey])
		assert.Equal(t, fmt.Sprintf("%d", livenessProbePeriod), annotations[daprLivenessProbePeriodKey])
		assert.Equal(t, fmt.Sprintf("%d", livenessProbeThreshold), annotations[daprLivenessProbeThresholdKey])
		assert.Equal(t, fmt.Sprintf("%d", readinessProbeDelay), annotations[daprReadinessProbeDelayKey])
		assert.Equal(t, fmt.Sprintf("%d", readinessProbeTimeout), annotations[daprReadinessProbeTimeoutKey])
		assert.Equal(t, fmt.Sprintf("%d", readinessProbePeriod), annotations[daprReadinessProbePeriodKey])
		assert.Equal(t, fmt.Sprintf("%d", readinessProbeThreshold), annotations[daprReadinessProbeThresholdKey])
		assert.Equal(t, daprImage, annotations[daprImageKey])
		assert.Equal(t, "true", annotations[daprAppSSLKey])
		assert.Equal(t, fmt.Sprintf("%d", maxRequestBodySize), annotations[daprMaxRequestBodySizeKey])
		assert.Equal(t, fmt.Sprintf("%d", readBufferSize), annotations[daprReadBufferSizeKey])
		assert.Equal(t, "true", annotations[daprHTTPStreamRequestBodyKey])
		assert.Equal(t, fmt.Sprintf("%d", gracefulShutdownSeconds), annotations[daprGracefulShutdownSecondsKey])
	})
}
