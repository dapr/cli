/*
Copyright 2026 The Dapr Authors
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
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/dapr/durabletask-go/api/protos"
)

func TestTimerOriginString(t *testing.T) {
	tests := []struct {
		name   string
		event  *protos.TimerCreatedEvent
		expect string
	}{
		{
			name: "nil origin",
			event: &protos.TimerCreatedEvent{
				FireAt: timestamppb.Now(),
			},
			expect: "",
		},
		{
			name: "createTimer",
			event: &protos.TimerCreatedEvent{
				FireAt: timestamppb.Now(),
				Origin: &protos.TimerCreatedEvent_CreateTimer{
					CreateTimer: &protos.TimerOriginCreateTimer{},
				},
			},
			expect: "createTimer",
		},
		{
			name: "externalEvent",
			event: &protos.TimerCreatedEvent{
				FireAt: timestamppb.Now(),
				Origin: &protos.TimerCreatedEvent_ExternalEvent{
					ExternalEvent: &protos.TimerOriginExternalEvent{
						Name: "myEvent",
					},
				},
			},
			expect: "externalEvent(myEvent)",
		},
		{
			name: "activityRetry",
			event: &protos.TimerCreatedEvent{
				FireAt: timestamppb.Now(),
				Origin: &protos.TimerCreatedEvent_ActivityRetry{
					ActivityRetry: &protos.TimerOriginActivityRetry{
						TaskExecutionId: "exec-123",
					},
				},
			},
			expect: "activityRetry(exec-123)",
		},
		{
			name: "childWorkflowRetry",
			event: &protos.TimerCreatedEvent{
				FireAt: timestamppb.Now(),
				Origin: &protos.TimerCreatedEvent_ChildWorkflowRetry{
					ChildWorkflowRetry: &protos.TimerOriginChildWorkflowRetry{
						InstanceId: "wf-456",
					},
				},
			},
			expect: "childWorkflowRetry(wf-456)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := timerOriginString(tc.event)
			assert.Equal(t, tc.expect, got)
		})
	}
}

func TestDeriveDetails_TimerCreated(t *testing.T) {
	first := &protos.HistoryEvent{
		Timestamp: timestamppb.Now(),
	}

	tests := []struct {
		name     string
		event    *protos.HistoryEvent
		contains []string
		excludes []string
	}{
		{
			name: "timer with no origin",
			event: &protos.HistoryEvent{
				EventType: &protos.HistoryEvent_TimerCreated{
					TimerCreated: &protos.TimerCreatedEvent{
						FireAt: timestamppb.Now(),
					},
				},
			},
			contains: []string{"fireAt="},
			excludes: []string{"origin="},
		},
		{
			name: "timer with createTimer origin",
			event: &protos.HistoryEvent{
				EventType: &protos.HistoryEvent_TimerCreated{
					TimerCreated: &protos.TimerCreatedEvent{
						FireAt: timestamppb.Now(),
						Origin: &protos.TimerCreatedEvent_CreateTimer{
							CreateTimer: &protos.TimerOriginCreateTimer{},
						},
					},
				},
			},
			contains: []string{"fireAt=", "origin=createTimer"},
		},
		{
			name: "timer with activityRetry origin",
			event: &protos.HistoryEvent{
				EventType: &protos.HistoryEvent_TimerCreated{
					TimerCreated: &protos.TimerCreatedEvent{
						FireAt: timestamppb.Now(),
						Origin: &protos.TimerCreatedEvent_ActivityRetry{
							ActivityRetry: &protos.TimerOriginActivityRetry{
								TaskExecutionId: "exec-abc",
							},
						},
					},
				},
			},
			contains: []string{"fireAt=", "origin=activityRetry(exec-abc)"},
		},
		{
			name: "timer with rerunParent and origin",
			event: &protos.HistoryEvent{
				EventType: &protos.HistoryEvent_TimerCreated{
					TimerCreated: &protos.TimerCreatedEvent{
						FireAt: timestamppb.Now(),
						RerunParentInstanceInfo: &protos.RerunParentInstanceInfo{
							InstanceID: "parent-123",
						},
						Origin: &protos.TimerCreatedEvent_ExternalEvent{
							ExternalEvent: &protos.TimerOriginExternalEvent{
								Name: "approval",
							},
						},
					},
				},
			},
			contains: []string{"fireAt=", "rerunParent=parent-123", "origin=externalEvent(approval)"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			details := deriveDetails(first, tc.event)
			assert.NotNil(t, details)
			for _, s := range tc.contains {
				assert.Contains(t, *details, s)
			}
			for _, s := range tc.excludes {
				assert.NotContains(t, *details, s)
			}
		})
	}
}
