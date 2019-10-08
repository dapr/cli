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

func PublishTopic(topic, payload string) error {
	if topic == "" {
		return errors.New("topic is missing")
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

	url := fmt.Sprintf("http://localhost:%s/v%s/publish/%s", fmt.Sprintf("%v", app.HTTPPort), api.RuntimeAPIVersion, topic)
	_, err = http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	return nil
}
