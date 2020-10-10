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

// Client is the interface the wraps all the methods exposed by the Dapr CLI.
type Client interface {
	// Get is used to invoke a method on a Dapr application with GET verb.
	Get(appID, method string) (string, error)
	// Post is used to invoke a method on a Dapr application with POST verb.
	Post(appID, method, payload string) (string, error)
	// Publish is used to publish event to a topic in a pubsub.
	Publish(topic, payload, pubsubName string) error
}

type Standalone struct {
	process DaprProcess
}

func NewClient() Client {
	return &Standalone{process: &daprProcess{}}
}
