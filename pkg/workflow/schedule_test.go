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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/durabletask-go/workflow"
	"github.com/dapr/kit/ptr"
)

func TestScheduleStatus(t *testing.T) {
	start := time.Date(2026, 10, 26, 9, 55, 0, 0, time.UTC)
	run := func(s protos.OrchestrationStatus, took time.Duration) *workflow.WorkflowScheduleRun {
		return &workflow.WorkflowScheduleRun{
			InstanceId:           "x",
			RuntimeStatus:        ptr.Of(s),
			CreatedTimestamp:     timestamppb.New(start),
			LastUpdatedTimestamp: timestamppb.New(start.Add(took)),
		}
	}
	tests := map[string]struct {
		st   *workflow.WorkflowScheduleStatus
		want string
	}{
		"no run yet": {&workflow.WorkflowScheduleStatus{}, "-"},
		"run purged": {&workflow.WorkflowScheduleStatus{LastRun: &workflow.WorkflowScheduleRun{InstanceId: "x"}}, "-"},
		"ok": {
			&workflow.WorkflowScheduleStatus{LastRun: run(protos.OrchestrationStatus_ORCHESTRATION_STATUS_COMPLETED, 9*time.Minute+8*time.Second)},
			"ok 9m8s",
		},
		"failed run": {
			&workflow.WorkflowScheduleStatus{LastRun: run(protos.OrchestrationStatus_ORCHESTRATION_STATUS_FAILED, 2*time.Second)},
			"failed 2s",
		},
		"sub-second": {
			&workflow.WorkflowScheduleStatus{LastRun: run(protos.OrchestrationStatus_ORCHESTRATION_STATUS_COMPLETED, 37*time.Millisecond+400*time.Microsecond)},
			"ok 37ms",
		},
		"running": {
			&workflow.WorkflowScheduleStatus{LastRun: run(protos.OrchestrationStatus_ORCHESTRATION_STATUS_RUNNING, 0)},
			"running",
		},
		"completed schedule keeps its last run": {
			&workflow.WorkflowScheduleStatus{
				State:   workflow.WorkflowScheduleStateCompleted,
				LastRun: run(protos.OrchestrationStatus_ORCHESTRATION_STATUS_COMPLETED, time.Second),
			},
			"ok 1s",
		},
		"completed schedule with no run read": {&workflow.WorkflowScheduleStatus{State: workflow.WorkflowScheduleStateCompleted}, "completed"},
		"failed schedule": {
			&workflow.WorkflowScheduleStatus{State: workflow.WorkflowScheduleStateFailed, FailureMessage: wrapperspb.String("bad spec\nmore")},
			"failed: bad spec",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.want, scheduleStatus(test.st))
		})
	}
}

func TestScheduleResultAndRuns(t *testing.T) {
	assert.Equal(t, "overlays/claude-code: update", scheduleResult(&workflow.WorkflowScheduleRun{
		Output: wrapperspb.String(`"overlays/claude-code: update\nsecond line"`),
	}))
	assert.Equal(t, "boom", scheduleResult(&workflow.WorkflowScheduleRun{
		Output:         wrapperspb.String(`"ignored"`),
		FailureDetails: &protos.TaskFailureDetails{ErrorMessage: "boom\nstack"},
	}))
	long := scheduleResult(&workflow.WorkflowScheduleRun{Output: wrapperspb.String(strings.Repeat("a", 100))})
	assert.Len(t, []rune(long), scheduleResultLength)
	assert.Empty(t, scheduleResult(nil))

	assert.Equal(t, "peach-agents/bump(claude-code)", scheduleRuns(&workflow.WorkflowSchedule{
		WorkflowName: "bump", TargetAppID: ptr.Of("peach-agents"), Input: wrapperspb.String(`"claude-code"`),
	}))
	assert.Equal(t, "bump", scheduleRuns(&workflow.WorkflowSchedule{WorkflowName: "bump"}))
}

func TestJSONInput(t *testing.T) {
	assert.Equal(t, `{"a":1}`, jsonInput(`{"a":1}`))
	assert.Equal(t, `42`, jsonInput(`42`))
	assert.Equal(t, `"claude-code"`, jsonInput(`claude-code`))
}

func TestFilterInstanceIDs(t *testing.T) {
	ids := []string{"a", "nightly_1", "dapr-internal-schedule-nightly", "nightlyX"}
	assert.Equal(t, ids, filterInstanceIDs(append([]string(nil), ids...), ListOptions{}), "schedule workflows are listed like any other")
	assert.Equal(t, []string{"nightly_1"}, filterInstanceIDs(append([]string(nil), ids...), ListOptions{InstanceIDPrefix: "nightly_"}))
}

func TestCreateScheduleOptions(t *testing.T) {
	request := func(opts CreateScheduleOptions) *protos.CreateWorkflowScheduleRequest {
		req := &protos.CreateWorkflowScheduleRequest{Schedule: new(protos.WorkflowSchedule)}
		for _, o := range createScheduleOptions(opts) {
			assert.NoError(t, o(req))
		}
		return req
	}

	assert.Nil(t, request(CreateScheduleOptions{}).GetSchedule().SkipMissed, "catching up is the default")
	assert.True(t, request(CreateScheduleOptions{SkipMissed: true}).GetSchedule().GetSkipMissed())
	assert.Equal(t, `"claude-code"`, request(CreateScheduleOptions{Input: ptr.Of("claude-code")}).GetSchedule().GetInput().GetValue())
}
