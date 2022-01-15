package kubernetes

// InjectOptions configure the injection behavior.
type InjectOptions struct {
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

type InjectOption func(*InjectOptions)

func NewInjectorOptions(opts ...InjectOption) InjectOptions {
	config := InjectOptions{} // nolint:exhaustivestruct
	for _, opt := range opts {
		opt(&config)
	}
	return config
}

func WithAppID(appID string) InjectOption {
	return func(config *InjectOptions) {
		config.appID = &appID
	}
}

func WithMetricsEnabled() InjectOption {
	return func(config *InjectOptions) {
		enabled := true
		config.metricsEnabled = &enabled
	}
}

func WithMetricsPort(port int) InjectOption {
	return func(config *InjectOptions) {
		config.metricsPort = &port
	}
}

func WithAppPort(port int) InjectOption {
	return func(config *InjectOptions) {
		config.appPort = &port
	}
}

func WithConfig(cfg string) InjectOption {
	return func(config *InjectOptions) {
		config.config = &cfg
	}
}

func WithAppProtocol(protocol string) InjectOption {
	return func(config *InjectOptions) {
		config.appProtocol = &protocol
	}
}

func WithProfileEnabled() InjectOption {
	return func(config *InjectOptions) {
		enabled := true
		config.profileEnabled = &enabled
	}
}

func WithLogLevel(logLevel string) InjectOption {
	return func(config *InjectOptions) {
		config.logLevel = &logLevel
	}
}

func WithAPITokenSecret(apiTokenSecret string) InjectOption {
	return func(config *InjectOptions) {
		config.apiTokenSecret = &apiTokenSecret
	}
}

func WithAppTokenSecret(appTokenSecret string) InjectOption {
	return func(config *InjectOptions) {
		config.appTokenSecret = &appTokenSecret
	}
}

func WithLogAsJSON() InjectOption {
	return func(config *InjectOptions) {
		enabled := true
		config.logAsJSON = &enabled
	}
}

func WithAppMaxConcurrency(maxConcurrency int) InjectOption {
	return func(config *InjectOptions) {
		config.appMaxConcurrency = &maxConcurrency
	}
}

func WithDebugEnabled() InjectOption {
	return func(config *InjectOptions) {
		enabled := true
		config.debugEnabled = &enabled
	}
}

func WithDebugPort(debugPort int) InjectOption {
	return func(config *InjectOptions) {
		config.debugPort = &debugPort
	}
}

func WithEnv(env string) InjectOption {
	return func(config *InjectOptions) {
		config.env = &env
	}
}

func WithCPULimit(cpuLimit string) InjectOption {
	return func(config *InjectOptions) {
		config.cpuLimit = &cpuLimit
	}
}

func WithMemoryLimit(memoryLimit string) InjectOption {
	return func(config *InjectOptions) {
		config.memoryLimit = &memoryLimit
	}
}

func WithCPURequest(cpuRequest string) InjectOption {
	return func(config *InjectOptions) {
		config.cpuRequest = &cpuRequest
	}
}

func WithMemoryRequest(memoryRequest string) InjectOption {
	return func(config *InjectOptions) {
		config.memoryRequest = &memoryRequest
	}
}

func WithListenAddresses(listenAddresses string) InjectOption {
	return func(config *InjectOptions) {
		config.listenAddresses = &listenAddresses
	}
}

func WithLivenessProbeDelay(livenessProbeDelay int) InjectOption {
	return func(config *InjectOptions) {
		config.livenessProbeDelay = &livenessProbeDelay
	}
}

func WithLivenessProbeTimeout(livenessProbeTimeout int) InjectOption {
	return func(config *InjectOptions) {
		config.livenessProbeTimeout = &livenessProbeTimeout
	}
}

func WithLivenessProbePeriod(livenessProbePeriod int) InjectOption {
	return func(config *InjectOptions) {
		config.livenessProbePeriod = &livenessProbePeriod
	}
}

func WithLivenessProbeThreshold(livenessProbeThreshold int) InjectOption {
	return func(config *InjectOptions) {
		config.livenessProbeThreshold = &livenessProbeThreshold
	}
}

func WithReadinessProbeDelay(readinessProbeDelay int) InjectOption {
	return func(config *InjectOptions) {
		config.readinessProbeDelay = &readinessProbeDelay
	}
}

func WithReadinessProbeTimeout(readinessProbeTimeout int) InjectOption {
	return func(config *InjectOptions) {
		config.readinessProbeTimeout = &readinessProbeTimeout
	}
}

func WithReadinessProbePeriod(readinessProbePeriod int) InjectOption {
	return func(config *InjectOptions) {
		config.readinessProbePeriod = &readinessProbePeriod
	}
}

func WithReadinessProbeThreshold(readinessProbeThreshold int) InjectOption {
	return func(config *InjectOptions) {
		config.readinessProbeThreshold = &readinessProbeThreshold
	}
}

func WithDaprImage(image string) InjectOption {
	return func(config *InjectOptions) {
		config.image = &image
	}
}

func WithAppSSL() InjectOption {
	return func(config *InjectOptions) {
		enabled := true
		config.appSSL = &enabled
	}
}

func WithMaxRequestBodySize(maxRequestBodySize int) InjectOption {
	return func(config *InjectOptions) {
		config.maxRequestBodySize = &maxRequestBodySize
	}
}

func WithReadBufferSize(readBufferSize int) InjectOption {
	return func(config *InjectOptions) {
		config.readBufferSize = &readBufferSize
	}
}

func WithHTTPStreamRequestBody() InjectOption {
	return func(config *InjectOptions) {
		enabled := true
		config.httpStreamRequestBody = &enabled
	}
}

func WithGracefulShutdownSeconds(gracefulShutdownSeconds int) InjectOption {
	return func(config *InjectOptions) {
		config.gracefulShutdownSeconds = &gracefulShutdownSeconds
	}
}
