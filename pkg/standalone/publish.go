// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/dapr/cli/pkg/api"
	"github.com/dapr/cli/utils"
)

// Publish publishes payload to topic in pubsub referenced by pubsubName.
func (s *Standalone) Publish(publishAppID, pubsubName, topic string, payload []byte, socket string) error {
	if publishAppID == "" {
		return errors.New("publishAppID is missing")
	}

	if pubsubName == "" {
		return errors.New("pubsubName is missing")
	}

	if topic == "" {
		return errors.New("topic is missing")
	}

	l, err := s.process.List()
	if err != nil {
		return err
	}

	instance, err := getDaprInstance(l, publishAppID)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://unix/v%s/publish/%s/%s", api.RuntimeAPIVersion, pubsubName, topic)

	var httpc http.Client
	if socket != "" {
		httpc.Transport = &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", utils.GetSocket(socket, publishAppID, "http"))
			},
		}
	} else {
		url = fmt.Sprintf("http://localhost:%s/v%s/publish/%s/%s", fmt.Sprintf("%v", instance.HTTPPort), api.RuntimeAPIVersion, pubsubName, topic)
	}

	r, err := httpc.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode >= 300 || r.StatusCode < 200 {
		return fmt.Errorf("unexpected status code %d on publishing to %s in %s", r.StatusCode, topic, pubsubName)
	}

	return nil
}

func getDaprInstance(list []ListOutput, publishAppID string) (ListOutput, error) {
	for i := 0; i < len(list); i++ {
		if list[i].AppID == publishAppID {
			return list[i], nil
		}
	}
	return ListOutput{}, errors.New("couldn't find a running Dapr instance")
}
