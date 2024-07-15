package standalone

import (
	"errors"
	"github.com/dapr/cli/pkg/metadata"
)

// ListOutput represents the application ID, application port and creation time.
type SubscriptionsOutput struct {
	AppID      string   `csv:"APP ID" json:"appId" yaml:"appId"`
	Topic      string   `csv:"TOPIC" json:"topic"  yaml:"topic"`
	PubSubName string   `csv:"PUBSUBNAME" json:"pubsubname"  yaml:"pubsubname"`
	Paths      []string `csv:"PATHS" json:"paths"  yaml:"paths"`
}

// Stop terminates the application process.
func Subscriptions(appID string) ([]SubscriptionsOutput, error) {
	l, err := List()
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, errors.New("no running Dapr sidecars found")
	}

	instance, err := getDaprInstance(l, appID)
	if err != nil {
		return nil, err
	}

	m, err := metadata.Get(instance.HTTPPort, instance.AppID, "")

	if err != nil {
		return nil, err
	}

	var output []SubscriptionsOutput

	for _, sub := range m.Subscriptions {
		o := SubscriptionsOutput{
			AppID:      appID,
			Topic:      sub.Topic,
			PubSubName: sub.PubSubName,
		}

		for _, r := range sub.Rules {
			o.Paths = append(o.Paths, r.Path)
		}

		output = append(output, o)
	}

	return output, nil
}
