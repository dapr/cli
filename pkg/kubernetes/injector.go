package kubernetes

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/dapr/dapr/pkg/injector"
	jsonpatch "github.com/evanphx/json-patch"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlDecoder "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

const (
	// Dapr annotation keys.
	daprEnabledKey                 = "dapr.io/enabled"
	daprAppPortKey                 = "dapr.io/app-port"
	daprConfigKey                  = "dapr.io/config"
	daprAppProtocolKey             = "dapr.io/app-protocol"
	daprAppIDKey                   = "dapr.io/app-id"
	daprEnableProfilingKey         = "dapr.io/enable-profiling"
	daprLogLevelKey                = "dapr.io/log-level"
	daprAPITokenSecretKey          = "dapr.io/api-token-secret" /* #nosec */
	daprAppTokenSecretKey          = "dapr.io/app-token-secret" /* #nosec */
	daprLogAsJSONKey               = "dapr.io/log-as-json"
	daprAppMaxConcurrencyKey       = "dapr.io/app-max-concurrency"
	daprEnableMetricsKey           = "dapr.io/enable-metrics"
	daprMetricsPortKey             = "dapr.io/metrics-port"
	daprEnableDebugKey             = "dapr.io/enable-debug"
	daprDebugPortKey               = "dapr.io/debug-port"
	daprEnvKey                     = "dapr.io/env"
	daprCPULimitKey                = "dapr.io/sidecar-cpu-limit"
	daprMemoryLimitKey             = "dapr.io/sidecar-memory-limit"
	daprCPURequestKey              = "dapr.io/sidecar-cpu-request"
	daprMemoryRequestKey           = "dapr.io/sidecar-memory-request"
	daprListenAddressesKey         = "dapr.io/sidecar-listen-addresses"
	daprLivenessProbeDelayKey      = "dapr.io/sidecar-liveness-probe-delay-seconds"
	daprLivenessProbeTimeoutKey    = "dapr.io/sidecar-liveness-probe-timeout-seconds"
	daprLivenessProbePeriodKey     = "dapr.io/sidecar-liveness-probe-period-seconds"
	daprLivenessProbeThresholdKey  = "dapr.io/sidecar-liveness-probe-threshold"
	daprReadinessProbeDelayKey     = "dapr.io/sidecar-readiness-probe-delay-seconds"
	daprReadinessProbeTimeoutKey   = "dapr.io/sidecar-readiness-probe-timeout-seconds"
	daprReadinessProbePeriodKey    = "dapr.io/sidecar-readiness-probe-period-seconds"
	daprReadinessProbeThresholdKey = "dapr.io/sidecar-readiness-probe-threshold"
	daprImageKey                   = "dapr.io/sidecar-image"
	daprAppSSLKey                  = "dapr.io/app-ssl"
	daprMaxRequestBodySizeKey      = "dapr.io/http-max-request-size"
	daprReadBufferSizeKey          = "dapr.io/http-read-buffer-size"
	daprHTTPStreamRequestBodyKey   = "dapr.io/http-stream-request-body"
	daprGracefulShutdownSecondsKey = "dapr.io/graceful-shutdown-seconds"

	// K8s kinds.
	pod         = "pod"
	deployment  = "deployment"
	replicaset  = "replicaset"
	daemonset   = "daemonset"
	statefulset = "statefulset"
	cronjob     = "cronjob"
	job         = "job"
	list        = "list"
)

type Injector interface {
	Inject(io.Reader, io.Writer) error
}

type K8sInjector struct {
	config   K8sInjectorConfig
	injected bool
}

type K8sInjectorConfig struct {
	// If TargetResource is set, we will search for it and then inject
	// annotations on that target resource. If it is not set, we will
	// update the first appropriate resource we find.
	TargetResource *string
}

func NewK8sInjector(config K8sInjectorConfig) *K8sInjector {
	return &K8sInjector{
		config: config,
	}
}

func (p *K8sInjector) Inject(inputs []io.Reader, out io.Writer, opts InjectOptions) error {
	for _, input := range inputs {
		err := p.processInput(input, out, opts)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *K8sInjector) processInput(input io.Reader, out io.Writer, opts InjectOptions) error {
	reader := yamlDecoder.NewYAMLReader(bufio.NewReaderSize(input, 4096))

	iterations := 0
	// Read from input and process until EOF or error.
	for {
		bytes, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// determine type first so that subsequent unmarshal can use correct version.
		var metaType metav1.TypeMeta
		if err = yaml.Unmarshal(bytes, &metaType); err != nil {
			return err
		}

		kind := strings.ToLower(metaType.Kind)
		if kind == list {
			var sourceList corev1.List
			if err = yaml.Unmarshal(bytes, &sourceList); err != nil {
				return err
			}
			items := []runtime.RawExtension{}
			for _, item := range sourceList.Items {
				var processedYAML []byte
				if p.injected {
					processedYAML = item.Raw
				} else {
					var injected bool
					processedYAML, injected, err = p.injectYAML(item.Raw, opts)
					if err != nil {
						return err
					}
					// Record that we have injected into a document.
					p.injected = injected
				}

				injectedJSON, err := yaml.YAMLToJSON(processedYAML)
				if err != nil {
					return err
				}
				items = append(items, runtime.RawExtension{Raw: injectedJSON})
			}
			sourceList.Items = items
			result, err := yaml.Marshal(sourceList)
			if err != nil {
				return err
			}

			if iterations > 0 {
				out.Write([]byte("---\n")) // WARN: Will leave trailing separator
			}
			_, err = out.Write(result)
			if err != nil {
				return err
			}

			iterations++
		} else {
			var processedYAML []byte
			if p.injected {
				processedYAML = bytes // We've already injected during this run so ingore the rest.
			} else {
				var injected bool
				processedYAML, injected, err = p.injectYAML(bytes, opts)
				if err != nil {
					return err
				}
				// Record that we have injected into a document.
				p.injected = injected
			}

			// Insert separator between documents.
			if iterations > 0 {
				out.Write([]byte("---\n"))
			}
			_, err = out.Write(processedYAML)
			if err != nil {
				return err
			}

			iterations++
		}
	}

	return nil
}

func (p *K8sInjector) injectYAML(input []byte, config InjectOptions) ([]byte, bool, error) {
	// We read the metadata again here in case we have extracted a sub resource from a list.
	var metaType metav1.TypeMeta
	if err := yaml.Unmarshal(input, &metaType); err != nil {
		return nil, false, err
	}

	var path string
	var annotations map[string]string
	var name string
	switch strings.ToLower(metaType.Kind) {
	case pod:
		pod := &corev1.Pod{}
		if err := yaml.Unmarshal(input, pod); err != nil {
			return nil, false, err
		}
		name = pod.Name
		annotations = pod.Annotations
		path = "/metadata/annotations"
	case cronjob:
		cronjob := &batchv1beta1.CronJob{}
		if err := yaml.Unmarshal(input, cronjob); err != nil {
			return nil, false, err
		}
		name = cronjob.Name
		annotations = cronjob.Spec.JobTemplate.Spec.Template.Annotations
		path = "/spec/jobTemplate/spec/template/metadata/annotations"
	case deployment:
		deployment := &appsv1.Deployment{}
		if err := yaml.Unmarshal(input, deployment); err != nil {
			return nil, false, err
		}
		name = deployment.Name
		annotations = deployment.Spec.Template.Annotations
		path = "/spec/template/metadata/annotations"
	case replicaset:
		replicaset := &appsv1.ReplicaSet{}
		if err := yaml.Unmarshal(input, replicaset); err != nil {
			return nil, false, err
		}
		name = replicaset.Name
		annotations = replicaset.Spec.Template.Annotations
		path = "/spec/template/metadata/annotations"
	case job:
		job := &batchv1.Job{}
		if err := yaml.Unmarshal(input, job); err != nil {
			return nil, false, err
		}
		name = job.Name
		annotations = job.Spec.Template.Annotations
		path = "/spec/template/metadata/annotations"
	case statefulset:
		statefulset := &appsv1.StatefulSet{}
		if err := yaml.Unmarshal(input, statefulset); err != nil {
			return nil, false, err
		}
		name = statefulset.Name
		annotations = statefulset.Spec.Template.Annotations
		path = "/spec/template/metadata/annotations"
	case daemonset:
		daemonset := &appsv1.DaemonSet{}
		if err := yaml.Unmarshal(input, daemonset); err != nil {
			return nil, false, err
		}
		name = daemonset.Name
		annotations = daemonset.Spec.Template.Annotations
		path = "/spec/template/metadata/annotations"
	default:
		// No injection needed for this kind.
		return input, false, nil
	}

	// TODO: Figure out a better way to do this
	if p.config.TargetResource != nil && *p.config.TargetResource != "" {
		if !strings.EqualFold(*p.config.TargetResource, name) {
			return input, false, nil
		}
	}

	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Get configured dapr annotations.
	daprAnnotations := getDaprAnnotations(&config)

	// Add dapr annotations to input document annotations.
	for k, v := range daprAnnotations {
		annotations[k] = v
	}

	// Create a patch operation for annotations.
	patchOps := []injector.PatchOperation{}
	patchOps = append(patchOps, injector.PatchOperation{
		Op:    "add",
		Path:  path,
		Value: annotations,
	})
	patchBytes, err := json.Marshal(patchOps)
	if err != nil {
		return nil, false, err
	}
	if len(patchBytes) == 0 {
		return input, false, nil
	}
	patch, err := jsonpatch.DecodePatch(patchBytes)
	if err != nil {
		return nil, false, err
	}

	// Convert the input document to JSON, apply the JSON patch, and convert back to YAML.
	inputAsJSON, err := yaml.YAMLToJSON(input)
	if err != nil {
		return nil, false, err
	}
	injectedAsJSON, err := patch.Apply(inputAsJSON)
	if err != nil {
		return nil, false, err
	}
	injectedAsYAML, err := yaml.JSONToYAML(injectedAsJSON)
	if err != nil {
		return nil, false, err
	}

	return injectedAsYAML, true, nil
}

func getDaprAnnotations(config *InjectOptions) map[string]string {
	annotations := make(map[string]string)

	annotations[daprEnabledKey] = "true"
	if config.appID != nil {
		annotations[daprAppIDKey] = *config.appID
	}
	if config.metricsEnabled != nil {
		annotations[daprEnableMetricsKey] = strconv.FormatBool(*config.metricsEnabled)
	}
	if config.metricsPort != nil {
		annotations[daprMetricsPortKey] = strconv.FormatInt(int64(*config.metricsPort), 10)
	}
	if config.appPort != nil {
		annotations[daprAppPortKey] = strconv.FormatInt(int64(*config.appPort), 10)
	}
	if config.config != nil {
		annotations[daprConfigKey] = *config.config
	}
	if config.appProtocol != nil {
		annotations[daprAppProtocolKey] = *config.appProtocol
	}
	if config.profileEnabled != nil {
		annotations[daprEnableProfilingKey] = strconv.FormatBool(*config.profileEnabled)
	}
	if config.logLevel != nil {
		annotations[daprLogLevelKey] = *config.logLevel
	}
	if config.logAsJson != nil {
		annotations[daprLogAsJSONKey] = strconv.FormatBool(*config.logAsJson)
	}
	if config.apiTokenSecret != nil {
		annotations[daprAPITokenSecretKey] = *config.apiTokenSecret
	}
	if config.appTokenSecret != nil {
		annotations[daprAppTokenSecretKey] = *config.appTokenSecret
	}
	if config.appMaxConcurrency != nil {
		annotations[daprAppMaxConcurrencyKey] = strconv.FormatInt(int64(*config.appMaxConcurrency), 10)
	}
	if config.debugEnabled != nil {
		annotations[daprEnableDebugKey] = strconv.FormatBool(*config.debugEnabled)
	}
	if config.env != nil {
		annotations[daprEnvKey] = *config.env
	}
	if config.cpuLimit != nil {
		annotations[daprCPULimitKey] = *config.cpuLimit
	}
	if config.memoryLimit != nil {
		annotations[daprMemoryLimitKey] = *config.memoryLimit
	}
	if config.cpuRequest != nil {
		annotations[daprCPURequestKey] = *config.cpuRequest
	}
	if config.memoryRequest != nil {
		annotations[daprMemoryRequestKey] = *config.memoryRequest
	}
	if config.listenAddresses != nil {
		annotations[daprListenAddressesKey] = *config.listenAddresses
	}
	if config.livenessProbeDelay != nil {
		annotations[daprLivenessProbeDelayKey] = strconv.FormatInt(int64(*config.livenessProbeDelay), 10)
	}
	if config.livenessProbeTimeout != nil {
		annotations[daprLivenessProbeTimeoutKey] = strconv.FormatInt(int64(*config.livenessProbeTimeout), 10)
	}
	if config.livenessProbePeriod != nil {
		annotations[daprLivenessProbePeriodKey] = strconv.FormatInt(int64(*config.livenessProbePeriod), 10)
	}
	if config.livenessProbeThreshold != nil {
		annotations[daprLivenessProbeThresholdKey] = strconv.FormatInt(int64(*config.livenessProbeThreshold), 10)
	}
	if config.readinessProbeDelay != nil {
		annotations[daprReadinessProbeDelayKey] = strconv.FormatInt(int64(*config.readinessProbeDelay), 10)
	}
	if config.readinessProbeTimeout != nil {
		annotations[daprReadinessProbeTimeoutKey] = strconv.FormatInt(int64(*config.readinessProbeTimeout), 10)
	}
	if config.readinessProbePeriod != nil {
		annotations[daprReadinessProbePeriodKey] = strconv.FormatInt(int64(*config.readinessProbePeriod), 10)
	}
	if config.readinessProbeThreshold != nil {
		annotations[daprReadinessProbeThresholdKey] = strconv.FormatInt(int64(*config.readinessProbeThreshold), 10)
	}
	if config.image != nil {
		annotations[daprImageKey] = *config.image
	}
	if config.appSSL != nil {
		annotations[daprAppSSLKey] = strconv.FormatBool(*config.appSSL)
	}
	if config.maxRequestBodySize != nil {
		annotations[daprMaxRequestBodySizeKey] = strconv.FormatInt(int64(*config.maxRequestBodySize), 10)
	}
	if config.httpStreamRequestBody != nil {
		annotations[daprHTTPStreamRequestBodyKey] = strconv.FormatBool(*config.httpStreamRequestBody)
	}
	if config.gracefulShutdownSeconds != nil {
		annotations[daprGracefulShutdownSecondsKey] = strconv.FormatInt(int64(*config.gracefulShutdownSeconds), 10)
	}

	return annotations
}

func getResourceObjectAndGKV(in io.Reader) (runtime.Object, *schema.GroupVersionKind, error) {
	reader := yamlDecoder.NewYAMLReader(bufio.NewReaderSize(in, 4096))
	bytes, err := reader.Read()
	if err != nil {
		return nil, nil, err
	}

	return scheme.Codecs.UniversalDeserializer().Decode(bytes, nil, nil)
}

func getAnnotationsFromDeployment(deployment *appsv1.Deployment) (map[string]string, error) {
	if deployment.Spec.Template.ObjectMeta.Annotations == nil {
		deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	return deployment.Spec.Template.ObjectMeta.Annotations, nil
}

func getAnnotationsFromPod(pod *corev1.Pod) (map[string]string, error) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	return pod.Annotations, nil
}
