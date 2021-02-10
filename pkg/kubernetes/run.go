// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

// RunConfig represents the application configuration parameters.
type RunConfig struct {
	AppID         string
	AppPort       int
	HTTPPort      int
	GRPCPort      int
	CodeDirectory string
	Arguments     []string
	Image         string
}

// RunOutput represents the run output.
type RunOutput struct {
	Message string
}

// Run executes the application based on the run configuration.
func Run(config *RunConfig) (*RunOutput, error) {
	return nil, nil
}
