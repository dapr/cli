// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

type DaprProcess interface {
	List() ([]ListOutput, error)
}

type daprProcess struct {
}

type Client interface {
	DaprProcess
	InvokeGet(appID, method string) (string, error)
	InvokePost(appID, method, payload string) (string, error)
	Publish(topic, payload, pubsubName string) error
}

type Standalone struct {
	process DaprProcess
}

func NewStandaloneClient() Client {
	return &Standalone{process: &daprProcess{}}
}

func (s *Standalone) List() ([]ListOutput, error) {
	return s.process.List()
}
