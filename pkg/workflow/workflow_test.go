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
	"testing"
	"time"

	"github.com/dapr/durabletask-go/api/protos"
	daprclient "github.com/dapr/go-sdk/client"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mockActorStateGetter implements the ActorStateGetter interface for testing
type mockActorStateGetter struct {
	responses map[string]*mockResponse
	calls     []string
}

type mockResponse struct {
	data []byte
	err  error
}

func (m *mockActorStateGetter) GetActorState(ctx context.Context, req *daprclient.GetActorStateRequest) (*daprclient.GetActorStateResponse, error) {
	key := req.ActorType + ":" + req.ActorID + ":" + req.KeyName
	m.calls = append(m.calls, key)

	if resp, exists := m.responses[key]; exists {
		if resp.err != nil {
			return nil, resp.err
		}
		return &daprclient.GetActorStateResponse{Data: resp.data}, nil
	}

	// Default: return empty response (simulates not found)
	return &daprclient.GetActorStateResponse{Data: []byte{}}, nil
}

func (m *mockActorStateGetter) expectCall(actorType, actorID, keyName string, data []byte, err error) {
	if m.responses == nil {
		m.responses = make(map[string]*mockResponse)
	}
	key := actorType + ":" + actorID + ":" + keyName
	m.responses[key] = &mockResponse{data: data, err: err}
}

func createTestHistoryEventData(eventID int32) []byte {
	event := &protos.HistoryEvent{
		EventId:   eventID,
		Timestamp: timestamppb.New(time.Date(2025, 9, 5, 12, 0, 0, 0, time.UTC)),
		EventType: &protos.HistoryEvent_ExecutionStarted{
			ExecutionStarted: &protos.ExecutionStartedEvent{
				Name: "test-workflow",
			},
		},
	}
	data, _ := protojson.Marshal(event)
	return data
}

func TestFetchHistory(t *testing.T) {
	tests := []struct {
		name           string
		appID          string
		namespace      string
		instanceID     string
		setupMock      func(*mockActorStateGetter)
		expectedEvents int
		expectedError  string
	}{
		{
			name:       "successful fetch starting from index 0",
			appID:      "test-app",
			namespace:  "test-namespace",
			instanceID: "test-instance",
			setupMock: func(m *mockActorStateGetter) {
				actorType := "dapr.internal.test-namespace.test-app.workflow"

				// Return two events starting from index 0
				m.expectCall(actorType, "test-instance", "history-000000", createTestHistoryEventData(1), nil)
				m.expectCall(actorType, "test-instance", "history-000001", createTestHistoryEventData(2), nil)
				// No more events (empty data signals end)
				m.expectCall(actorType, "test-instance", "history-000002", []byte{}, nil)
			},
			expectedEvents: 2,
		},
		{
			name:       "successful fetch starting from index 1",
			appID:      "test-app",
			namespace:  "default",
			instanceID: "test-instance-1",
			setupMock: func(m *mockActorStateGetter) {
				actorType := "dapr.internal.default.test-app.workflow"

				// No event at index 0 (error), but event at index 1
				m.expectCall(actorType, "test-instance-1", "history-000000", nil, errors.New("not found"))
				m.expectCall(actorType, "test-instance-1", "history-000001", createTestHistoryEventData(1), nil)
				// No more events
				m.expectCall(actorType, "test-instance-1", "history-000002", []byte{}, nil)
			},
			expectedEvents: 1,
		},
		{
			name:       "no events found at all",
			appID:      "empty-app",
			namespace:  "default",
			instanceID: "empty-instance",
			setupMock: func(m *mockActorStateGetter) {
				actorType := "dapr.internal.default.empty-app.workflow"

				// No events at index 0 or 1
				m.expectCall(actorType, "empty-instance", "history-000000", nil, errors.New("not found"))
				m.expectCall(actorType, "empty-instance", "history-000001", nil, errors.New("not found"))
			},
			expectedEvents: 0,
		},
		{
			name:       "context timeout error",
			appID:      "timeout-app",
			namespace:  "default",
			instanceID: "timeout-instance",
			setupMock: func(m *mockActorStateGetter) {
				actorType := "dapr.internal.default.timeout-app.workflow"
				m.expectCall(actorType, "timeout-instance", "history-000000", nil, context.DeadlineExceeded)
			},
			expectedEvents: 0,
			expectedError:  "context deadline exceeded",
		},
		{
			name:       "context cancellation error",
			appID:      "cancel-app",
			namespace:  "default",
			instanceID: "cancel-instance",
			setupMock: func(m *mockActorStateGetter) {
				actorType := "dapr.internal.default.cancel-app.workflow"
				m.expectCall(actorType, "cancel-instance", "history-000000", nil, context.Canceled)
			},
			expectedEvents: 0,
			expectedError:  "context canceled",
		},
		{
			name:       "decode error",
			appID:      "decode-error-app",
			namespace:  "default",
			instanceID: "decode-error-instance",
			setupMock: func(m *mockActorStateGetter) {
				actorType := "dapr.internal.default.decode-error-app.workflow"
				// Return invalid data that can't be decoded
				m.expectCall(actorType, "decode-error-instance", "history-000000", []byte("invalid data"), nil)
			},
			expectedEvents: 0,
			expectedError:  "failed to decode history event history-000000",
		},
		{
			name:       "multiple events with gap - stops after finding some",
			appID:      "gap-app",
			namespace:  "default",
			instanceID: "gap-instance",
			setupMock: func(m *mockActorStateGetter) {
				actorType := "dapr.internal.default.gap-app.workflow"

				// Events at index 0 and 1, then error (should stop since we found events)
				m.expectCall(actorType, "gap-instance", "history-000000", createTestHistoryEventData(1), nil)
				m.expectCall(actorType, "gap-instance", "history-000001", createTestHistoryEventData(2), nil)
				m.expectCall(actorType, "gap-instance", "history-000002", nil, errors.New("not found"))
			},
			expectedEvents: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGetter := &mockActorStateGetter{}
			tt.setupMock(mockGetter)

			ctx := context.Background()
			events, err := fetchHistory(ctx, mockGetter, tt.appID, tt.namespace, tt.instanceID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Len(t, events, tt.expectedEvents)

				// Verify event content for successful cases
				for i, event := range events {
					assert.NotNil(t, event)
					assert.Equal(t, int32(i+1), event.EventId)
				}
			}
		})
	}
}

func TestDecodeHistoryEvent(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty data",
			input:       []byte{},
			expectError: true,
			errorMsg:    "empty value",
		},
		{
			name:        "valid protojson",
			input:       createTestHistoryEventData(1),
			expectError: false,
		},
		{
			name: "valid quoted json string",
			input: func() []byte {
				// Create a JSON string that contains protojson data
				eventData := createTestHistoryEventData(1)
				quotedJSON, _ := json.Marshal(string(eventData))
				return quotedJSON
			}(),
			expectError: false,
		},
		{
			name:        "invalid data",
			input:       []byte("completely invalid data"),
			expectError: true,
			errorMsg:    "unable to decode history event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := decodeHistoryEvent(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, event)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, event)
				assert.Equal(t, int32(1), event.EventId)
			}
		})
	}
}

func TestUnquoteJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expected    string
		expectError bool
	}{
		{
			name:        "valid quoted string",
			input:       []byte(`"hello world"`),
			expected:    "hello world",
			expectError: false,
		},
		{
			name:        "valid escaped json",
			input:       []byte(`"{\"key\":\"value\"}"`),
			expected:    `{"key":"value"}`,
			expectError: false,
		},
		{
			name:        "invalid json",
			input:       []byte(`invalid`),
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := unquoteJSON(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
