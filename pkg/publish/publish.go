package publish

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/actionscore/cli/pkg/standalone"
)

type messageEnvelope struct {
	Topic     string      `json:"topic,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	CreatedAt string      `json:"createdAt"`
}

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
			m := messageEnvelope{
				CreatedAt: time.Now().Format(time.RFC3339),
			}

			if payload != "" {
				var data interface{}
				err := json.Unmarshal([]byte(payload), &data)
				if err != nil {
					return err
				}

				m.Data = data
			}

			b, err := json.Marshal(&m)
			if err != nil {
				return err
			}

			_, err = http.Post(fmt.Sprintf("http://localhost:%s/invoke/%s", fmt.Sprintf("%v", lo.ActionsPort), topic), "application/json", bytes.NewBuffer(b))
			if err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("App id %s not found", appID)
}
