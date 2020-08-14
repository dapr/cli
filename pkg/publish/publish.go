// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package publish

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/dapr/cli/pkg/api"
	"github.com/dapr/cli/pkg/standalone"
)

// SendPayloadToTopic publishes the topic
func SendPayloadToTopic(topic, payload, pubsubName string) error {
	if topic == "" {
		return errors.New("topic is missing")
	}
	if pubsubName == "" {
		return errors.New("pubsubName is missing")
	}

	l, err := standalone.List()
	if err != nil {
		return err
	}

	if len(l) == 0 {
		return errors.New("couldn't find a running Dapr instance")
	}

	app := l[0]
	b := []byte{}

	if payload != "" {
		b = []byte(payload)
	}

	url := fmt.Sprintf("http://localhost:%s/v%s/publish/%s/%s", fmt.Sprintf("%v", app.HTTPPort), api.RuntimeAPIVersion, pubsubName, topic)
	// nolint: gosec
	r, err := http.Post(url, "application/json", bytes.NewBuffer(b))

	if r != nil {
		defer r.Body.Close()
	}

	if err != nil {
		return err
	}

	return nil
}
