// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/dapr/cli/pkg/api"
)

// Publish publishes payload to topic in pubsub referenced by pubsubName.
func (s *Standalone) Publish(publishAppID, pubsubName, topic string, payload []byte) error {
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
		return fmt.Errorf("error: %w", err)
	}

	daprHTTPPort, err := getDaprHTTPPort(l, publishAppID)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	url := fmt.Sprintf("http://localhost:%s/v%s/publish/%s/%s", fmt.Sprintf("%v", daprHTTPPort), api.RuntimeAPIVersion, pubsubName, topic)
	// nolint: gosec
	r, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}
	defer r.Body.Close()
	if r.StatusCode >= 300 || r.StatusCode < 200 {
		return fmt.Errorf("unexpected status code %d on publishing to %s in %s", r.StatusCode, topic, pubsubName)
	}

	return nil
}

func getDaprHTTPPort(list []ListOutput, publishAppID string) (int, error) {
	for i := 0; i < len(list); i++ {
		if list[i].AppID == publishAppID {
			return list[i].HTTPPort, nil
		}
	}
	return 0, errors.New("couldn't find a running Dapr instance")
}
