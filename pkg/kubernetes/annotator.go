package kubernetes

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	yamlDecoder "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"github.com/dapr/dapr/pkg/injector"
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

	cronjobAnnotationsPath  = "/spec/jobTemplate/spec/template/metadata/annotations"
	podAnnotationsPath      = "/metadata/annotations"
	templateAnnotationsPath = "/spec/template/metadata/annotations"
)

type Annotator interface {
	Annotate(io.Reader, io.Writer) error
}

type K8sAnnotator struct {
	config    K8sAnnotatorConfig
	annotated bool
}

type K8sAnnotatorConfig struct {
	// If TargetResource is set, we will search for it and then inject
	// annotations on that target resource. If it is not set, we will
	// update the first appropriate resource we find.
	TargetResource *string
	// If TargetNamespace is set, we will search for the target resource
	// in the provided target namespace. If it is not set, we will
	// just search for the first occurrence of the target resource.
	TargetNamespace *string
}

func NewK8sAnnotator(config K8sAnnotatorConfig) *K8sAnnotator {
	return &K8sAnnotator{
		config:    config,
		annotated: false,
	}
}

// Annotate injects dapr annotations into the kubernetes resource.
func (p *K8sAnnotator) Annotate(inputs []io.Reader, out io.Writer, opts AnnotateOptions) error {
	for _, input := range inputs {
		err := p.processInput(input, out, opts)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *K8sAnnotator) processInput(input io.Reader, out io.Writer, opts AnnotateOptions) error {
	reader := yamlDecoder.NewYAMLReader(bufio.NewReaderSize(input, 4096))

	var result []byte
	iterations := 0
	// Read from input and process until EOF or error.
	for {
		bytes, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		// Check if the input is a list as
		// these requires additional processing.
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
				processedYAML, err = p.processYAML(item.Raw, opts)
				if err != nil {
					return err
				}
				var annotatedJSON []byte
				annotatedJSON, err = yaml.YAMLToJSON(processedYAML)
				if err != nil {
					return err
				}
				items = append(items, runtime.RawExtension{Raw: annotatedJSON}) // nolint:exhaustivestruct
			}
			sourceList.Items = items
			result, err = yaml.Marshal(sourceList)
			if err != nil {
				return err
			}
		} else {
			var processedYAML []byte
			processedYAML, err = p.processYAML(bytes, opts)
			if err != nil {
				return err
			}
			result = processedYAML
		}

		// Insert separator between documents.
		if iterations > 0 {
			out.Write([]byte("---\n"))
		}

		// Write result from processing into the writer.
		_, err = out.Write(result)
		if err != nil {
			return err
		}

		iterations++
	}

	return nil
}

func (p *K8sAnnotator) processYAML(yamlBytes []byte, opts AnnotateOptions) ([]byte, error) {
	var err error
	var processedYAML []byte
	if p.annotated {
		// We can only inject dapr into a single resource per execution as the configuration
		// options are scoped to a single resource e.g. app-id, port, etc. are specific to a
		// dapr enabled resource. Instead we expect multiple runs to be piped together.
		processedYAML = yamlBytes
	} else {
		var annotated bool
		processedYAML, annotated, err = p.annotateYAML(yamlBytes, opts)
		if err != nil {
			return nil, err
		}
		if annotated {
			// Record that we have injected into a document.
			p.annotated = annotated
		}
	}
	return processedYAML, nil
}

func (p *K8sAnnotator) annotateYAML(input []byte, config AnnotateOptions) ([]byte, bool, error) {
	// We read the metadata again here so this method is encapsulated.
	var metaType metav1.TypeMeta
	if err := yaml.Unmarshal(input, &metaType); err != nil {
		return nil, false, err
	}

	// If the input resource is a 'kind' that
	// we want to inject dapr into, then we
	// Unmarshal the input into the appropriate
	// type and set the required fields to build
	// a patch (path, value, op).
	var path string
	var annotations map[string]string
	var name string
	var ns string

	kind := strings.ToLower(metaType.Kind)
	switch kind {
	case pod:
		pod := &corev1.Pod{} // nolint:exhaustivestruct
		if err := yaml.Unmarshal(input, pod); err != nil {
			return nil, false, err
		}
		name = pod.Name
		annotations = pod.Annotations
		path = podAnnotationsPath
		ns = getNamespaceOrDefault(pod)
	case cronjob:
		cronjob := &batchv1beta1.CronJob{} // nolint:exhaustivestruct
		if err := yaml.Unmarshal(input, cronjob); err != nil {
			return nil, false, err
		}
		name = cronjob.Name
		annotations = cronjob.Spec.JobTemplate.Spec.Template.Annotations
		path = cronjobAnnotationsPath
		ns = getNamespaceOrDefault(cronjob)
	case deployment:
		deployment := &appsv1.Deployment{} // nolint:exhaustivestruct
		if err := yaml.Unmarshal(input, deployment); err != nil {
			return nil, false, err
		}
		name = deployment.Name
		annotations = deployment.Spec.Template.Annotations
		path = templateAnnotationsPath
		ns = getNamespaceOrDefault(deployment)
	case replicaset:
		replicaset := &appsv1.ReplicaSet{} // nolint:exhaustivestruct
		if err := yaml.Unmarshal(input, replicaset); err != nil {
			return nil, false, err
		}
		name = replicaset.Name
		annotations = replicaset.Spec.Template.Annotations
		path = templateAnnotationsPath
		ns = getNamespaceOrDefault(replicaset)
	case job:
		job := &batchv1.Job{} // nolint:exhaustivestruct
		if err := yaml.Unmarshal(input, job); err != nil {
			return nil, false, err
		}
		name = job.Name
		annotations = job.Spec.Template.Annotations
		path = templateAnnotationsPath
		ns = getNamespaceOrDefault(job)
	case statefulset:
		statefulset := &appsv1.StatefulSet{} // nolint:exhaustivestruct
		if err := yaml.Unmarshal(input, statefulset); err != nil {
			return nil, false, err
		}
		name = statefulset.Name
		annotations = statefulset.Spec.Template.Annotations
		path = templateAnnotationsPath
		ns = getNamespaceOrDefault(statefulset)
	case daemonset:
		daemonset := &appsv1.DaemonSet{} // nolint:exhaustivestruct
		if err := yaml.Unmarshal(input, daemonset); err != nil {
			return nil, false, err
		}
		name = daemonset.Name
		annotations = daemonset.Spec.Template.Annotations
		path = templateAnnotationsPath
		ns = getNamespaceOrDefault(daemonset)
	default:
		// No annotation needed for this kind.
		return input, false, nil
	}

	// TODO: Currently this is where we decide not to
	// annotate dapr on this resource as it isn't the
	// target we are looking for. This is a bit late
	// so it would be good to find a earlier place to
	// do this.
	if p.config.TargetResource != nil && *p.config.TargetResource != "" {
		if !strings.EqualFold(*p.config.TargetResource, name) {
			// Not the resource name we're annotating.
			return input, false, nil
		}
		if p.config.TargetNamespace != nil && *p.config.TargetNamespace != "" {
			if !strings.EqualFold(*p.config.TargetNamespace, ns) {
				// Not the namespace we're annotating.
				return input, false, nil
			}
		}
	}

	// Get the dapr annotations and set them on the
	// resources existing annotation map. This will
	// override any existing conflicting annotations.
	if annotations == nil {
		annotations = make(map[string]string)
	}
	daprAnnotations := getDaprAnnotations(&config)
	for k, v := range daprAnnotations {
		// TODO: Should we log when we are overwriting?
		// if _, exists := annotations[k]; exists {}.
		annotations[k] = v
	}

	// Check if the app id has been set, if not, we'll
	// use the resource metadata namespace, kind and name.
	// For example: namespace-kind-name.
	if _, appIDSet := annotations[daprAppIDKey]; !appIDSet {
		annotations[daprAppIDKey] = fmt.Sprintf("%s-%s-%s", ns, kind, name)
	}

	// Create a patch operation for the annotations.
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

	// As we are applying the patch as a json patch,
	// we have to convert the current YAML resource to
	// JSON, apply the patch and then convert back.
	inputAsJSON, err := yaml.YAMLToJSON(input)
	if err != nil {
		return nil, false, err
	}
	annotatedAsJSON, err := patch.Apply(inputAsJSON)
	if err != nil {
		return nil, false, err
	}
	annotatedAsYAML, err := yaml.JSONToYAML(annotatedAsJSON)
	if err != nil {
		return nil, false, err
	}

	return annotatedAsYAML, true, nil
}

type NamespacedObject interface {
	GetNamespace() string
}

func getNamespaceOrDefault(obj NamespacedObject) string {
	ns := obj.GetNamespace()
	if ns == "" {
		return "default"
	}
	return ns
}

func getDaprAnnotations(config *AnnotateOptions) map[string]string {
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
	if config.logAsJSON != nil {
		annotations[daprLogAsJSONKey] = strconv.FormatBool(*config.logAsJSON)
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
	if config.debugPort != nil {
		annotations[daprDebugPortKey] = strconv.FormatInt(int64(*config.debugPort), 10)
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
	if config.readBufferSize != nil {
		annotations[daprReadBufferSizeKey] = strconv.FormatInt(int64(*config.readBufferSize), 10)
	}
	if config.httpStreamRequestBody != nil {
		annotations[daprHTTPStreamRequestBodyKey] = strconv.FormatBool(*config.httpStreamRequestBody)
	}
	if config.gracefulShutdownSeconds != nil {
		annotations[daprGracefulShutdownSecondsKey] = strconv.FormatInt(int64(*config.gracefulShutdownSeconds), 10)
	}

	return annotations
}
