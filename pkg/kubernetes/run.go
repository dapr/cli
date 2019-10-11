// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

// RunConfig to represent application configuration parameters
type RunConfig struct {
	AppID         string
	AppPort       int
	HTTPPort      int
	GRPCPort      int
	CodeDirectory string
	Arguments     []string
	Image         string
}

// RunOutput to represent the run output
type RunOutput struct {
	Message string
}

// Run based on run configuration
func Run(config *RunConfig) (*RunOutput, error) {
	return nil, nil
}
