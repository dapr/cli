/*
Copyright 2025 The Dapr Authors
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

package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dapr/durabletask-go/api/protos"
	daprclient "github.com/dapr/go-sdk/client"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const maxHistoryEntries = 1000

// Simplified interface for testing purposes
type ActorStateGetter interface {
	GetActorState(ctx context.Context, in *daprclient.GetActorStateRequest) (*daprclient.GetActorStateResponse, error)
}

func FetchHistory(ctx context.Context, appID, namespace, instanceID string) ([]*protos.HistoryEvent, error) {
	c, err := daprclient.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Dapr client: %w", err)
	}
	defer c.Close()

	return fetchHistory(ctx, c, appID, namespace, instanceID)
}

func fetchHistory(ctx context.Context, actorStateGetter ActorStateGetter, appID, namespace, instanceID string) ([]*protos.HistoryEvent, error) {
	actorType := fmt.Sprintf("dapr.internal.%s.%s.workflow", namespace, appID)
	var events []*protos.HistoryEvent

	// Try starting from index 0, then 1 if no events found at 0
	for startIndex := 0; startIndex <= 1; startIndex++ {
		if len(events) > 0 {
			break // Found events, no need to try next start index
		}

		for i := startIndex; i < maxHistoryEntries; i++ {
			key := fmt.Sprintf("history-%06d", i)

			req := &daprclient.GetActorStateRequest{
				ActorType: actorType,
				ActorID:   instanceID,
				KeyName:   key,
			}

			resp, err := actorStateGetter.GetActorState(ctx, req)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					return nil, err
				}
				// For other errors, stop enumeration
				break
			}

			if resp == nil || len(resp.Data) == 0 {
				// No more data, stop enumeration
				break
			}

			decoded, err := decodeHistoryEvent(resp.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to decode history event %s: %w", key, err)
			}

			events = append(events, decoded)
		}
	}

	return events, nil
}

func decodeHistoryEvent(data []byte) (*protos.HistoryEvent, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty value")
	}

	var event protos.HistoryEvent

	if err := protojson.Unmarshal(data, &event); err == nil {
		return &event, nil
	}

	if unquoted, err := unquoteJSON(data); err == nil {
		if err := protojson.Unmarshal([]byte(unquoted), &event); err == nil {
			return &event, nil
		}
	}

	if err := proto.Unmarshal(data, &event); err == nil {
		return &event, nil
	}

	return nil, fmt.Errorf("unable to decode history event (len=%d)", len(data))
}

func unquoteJSON(data []byte) (string, error) {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return "", err
	}
	return s, nil
}
