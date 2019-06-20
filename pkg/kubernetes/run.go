package kubernetes

type RunConfig struct {
	AppID         string
	AppPort       int
	Port          int
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
