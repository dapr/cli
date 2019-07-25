package publish

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/actionscore/cli/pkg/api"
	"github.com/actionscore/cli/pkg/standalone"
)

func PublishTopic(appID, topic, payload string) error {
	if topic == "" {
		return errors.New("topic is missing")
	} else if appID == "" {
		return errors.New("app id is missing")
	}

	l, err := standalone.List()
	if err != nil {
		return err
	}

	for _, lo := range l {
		if lo.AppID == appID {
			b := []byte{}

			if payload != "" {
				b = []byte(payload)
			}

			url := fmt.Sprintf("http://localhost:%s/v%s/publish/%s", fmt.Sprintf("%v", lo.ActionsPort), api.RuntimeAPIVersion, topic)
			_, err = http.Post(url, "application/json", bytes.NewBuffer(b))
			if err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("App id %s not found", appID)
}
