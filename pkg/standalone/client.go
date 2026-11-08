/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package standalone

type DaprProcess interface {
	List() ([]ListOutput, error)
}

type daprProcess struct{}

// Client is the interface the wraps all the methods exposed by the Dapr CLI.
type Client interface {
	// Invoke is a command to invoke a remote or local dapr instance.
	Invoke(appID, method string, data []byte, verb string, socket string) (string, error)
	// Publish is used to publish event to a topic in a pubsub for an app ID.
	Publish(publishAppID, pubsubName, topic string, payload []byte, socket string, metadata map[string]interface{}) error
}

type Standalone struct {
	process DaprProcess
}

func NewClient() Client {
	return &Standalone{process: &daprProcess{}}
}
