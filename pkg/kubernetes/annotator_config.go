package kubernetes

// AnnotateOptions configure the injection behavior.
type AnnotateOptions struct {
	appID                   *string
	metricsEnabled          *bool
	metricsPort             *int
	appPort                 *int
	config                  *string
	appProtocol             *string
	profileEnabled          *bool
	logLevel                *string
	apiTokenSecret          *string
	appTokenSecret          *string
	logAsJSON               *bool
	appMaxConcurrency       *int
	debugEnabled            *bool
	debugPort               *int
	env                     *string
	cpuLimit                *string
	memoryLimit             *string
	cpuRequest              *string
	memoryRequest           *string
	listenAddresses         *string
	livenessProbeDelay      *int
	livenessProbeTimeout    *int
	livenessProbePeriod     *int
	livenessProbeThreshold  *int
	readinessProbeDelay     *int
	readinessProbeTimeout   *int
	readinessProbePeriod    *int
	readinessProbeThreshold *int
	image                   *string
	appSSL                  *bool
	maxRequestBodySize      *int
	readBufferSize          *int
	httpStreamRequestBody   *bool
	gracefulShutdownSeconds *int
}

type AnnoteOption func(*AnnotateOptions)

func NewAnnotateOptions(opts ...AnnoteOption) AnnotateOptions {
	config := AnnotateOptions{} // nolint:exhaustivestruct
	for _, opt := range opts {
		opt(&config)
	}
	return config
}

func WithAppID(appID string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.appID = &appID
	}
}

func WithMetricsEnabled() AnnoteOption {
	return func(config *AnnotateOptions) {
		enabled := true
		config.metricsEnabled = &enabled
	}
}

func WithMetricsPort(port int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.metricsPort = &port
	}
}

func WithAppPort(port int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.appPort = &port
	}
}

func WithConfig(cfg string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.config = &cfg
	}
}

func WithAppProtocol(protocol string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.appProtocol = &protocol
	}
}

func WithProfileEnabled() AnnoteOption {
	return func(config *AnnotateOptions) {
		enabled := true
		config.profileEnabled = &enabled
	}
}

func WithLogLevel(logLevel string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.logLevel = &logLevel
	}
}

func WithAPITokenSecret(apiTokenSecret string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.apiTokenSecret = &apiTokenSecret
	}
}

func WithAppTokenSecret(appTokenSecret string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.appTokenSecret = &appTokenSecret
	}
}

func WithLogAsJSON() AnnoteOption {
	return func(config *AnnotateOptions) {
		enabled := true
		config.logAsJSON = &enabled
	}
}

func WithAppMaxConcurrency(maxConcurrency int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.appMaxConcurrency = &maxConcurrency
	}
}

func WithDebugEnabled() AnnoteOption {
	return func(config *AnnotateOptions) {
		enabled := true
		config.debugEnabled = &enabled
	}
}

func WithDebugPort(debugPort int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.debugPort = &debugPort
	}
}

func WithEnv(env string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.env = &env
	}
}

func WithCPULimit(cpuLimit string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.cpuLimit = &cpuLimit
	}
}

func WithMemoryLimit(memoryLimit string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.memoryLimit = &memoryLimit
	}
}

func WithCPURequest(cpuRequest string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.cpuRequest = &cpuRequest
	}
}

func WithMemoryRequest(memoryRequest string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.memoryRequest = &memoryRequest
	}
}

func WithListenAddresses(listenAddresses string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.listenAddresses = &listenAddresses
	}
}

func WithLivenessProbeDelay(livenessProbeDelay int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.livenessProbeDelay = &livenessProbeDelay
	}
}

func WithLivenessProbeTimeout(livenessProbeTimeout int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.livenessProbeTimeout = &livenessProbeTimeout
	}
}

func WithLivenessProbePeriod(livenessProbePeriod int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.livenessProbePeriod = &livenessProbePeriod
	}
}

func WithLivenessProbeThreshold(livenessProbeThreshold int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.livenessProbeThreshold = &livenessProbeThreshold
	}
}

func WithReadinessProbeDelay(readinessProbeDelay int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.readinessProbeDelay = &readinessProbeDelay
	}
}

func WithReadinessProbeTimeout(readinessProbeTimeout int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.readinessProbeTimeout = &readinessProbeTimeout
	}
}

func WithReadinessProbePeriod(readinessProbePeriod int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.readinessProbePeriod = &readinessProbePeriod
	}
}

func WithReadinessProbeThreshold(readinessProbeThreshold int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.readinessProbeThreshold = &readinessProbeThreshold
	}
}

func WithDaprImage(image string) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.image = &image
	}
}

func WithAppSSL() AnnoteOption {
	return func(config *AnnotateOptions) {
		enabled := true
		config.appSSL = &enabled
	}
}

func WithMaxRequestBodySize(maxRequestBodySize int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.maxRequestBodySize = &maxRequestBodySize
	}
}

func WithReadBufferSize(readBufferSize int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.readBufferSize = &readBufferSize
	}
}

func WithHTTPStreamRequestBody() AnnoteOption {
	return func(config *AnnotateOptions) {
		enabled := true
		config.httpStreamRequestBody = &enabled
	}
}

func WithGracefulShutdownSeconds(gracefulShutdownSeconds int) AnnoteOption {
	return func(config *AnnotateOptions) {
		config.gracefulShutdownSeconds = &gracefulShutdownSeconds
	}
}
