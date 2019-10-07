package kubernetes

type RunConfig struct {
	AppID         string
	AppPort       int
	HTTPPort      int
	GRPCPort      int
	CodeDirectory string
	Arguments     []string
	Image         string
}

type RunOutput struct {
	Message string
}

func Run(config *RunConfig) (*RunOutput, error) {
	return nil, nil
}
