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
	"fmt"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/dapr/cli/cmd/runtime"
	"github.com/dapr/cli/pkg/workflow/dclient"
	"github.com/dapr/cli/utils"
	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/durabletask-go/workflow"
	"github.com/dapr/kit/ptr"
)

type HistoryOptions struct {
	KubernetesMode bool
	Namespace      string
	AppID          string
	InstanceID     string
}

type HistoryOutputWide struct {
	Namespace   string    `csv:"-" json:"namespace,omitempty" yaml:"namespace,omitempty"`
	AppID       string    `csv:"-"    json:"appID"    yaml:"appID"`
	Play        int       `csv:"PLAY" json:"play" yaml:"play"`
	Type        string    `csv:"TYPE"      json:"type"      yaml:"type"`
	Name        *string   `csv:"NAME"      json:"name"      yaml:"name"`
	EventID     *int32    `csv:"EVENTID,omitempty"      json:"eventId,omitempty"      yaml:"eventId,omitempty"`
	Timestamp   time.Time `csv:"TIMESTAMP" json:"timestamp" yaml:"timestamp"`
	Elapsed     string    `csv:"ELAPSED" json:"elapsed" yaml:"elapsed"`
	Status      string    `csv:"STATUS"    json:"status"    yaml:"status"`
	Details     *string   `csv:"DETAILS"   json:"details"   yaml:"details"`
	Router      *string   `csv:"ROUTER,omitempty"       json:"router,omitempty"       yaml:"router,omitempty"`
	ExecutionID *string   `csv:"EXECUTION_ID,omitempty" json:"executionId,omitempty"  yaml:"executionId,omitempty"`

	Attrs *string `csv:"ATTRS,omitempty" json:"attrs,omitempty" yaml:"attrs,omitempty"`
}

type HistoryOutputShort struct {
	Type    string `csv:"TYPE"      json:"type"      yaml:"type"`
	Name    string `csv:"NAME"      json:"name"      yaml:"name"`
	EventID string `csv:"EVENTID,omitempty"      json:"eventId,omitempty"      yaml:"eventId,omitempty"`
	Elapsed string `csv:"ELAPSED" json:"elapsed" yaml:"elapsed"`
	Status  string `csv:"STATUS"    json:"status"    yaml:"status"`
	Details string `csv:"DETAILS"   json:"details"   yaml:"details"`
}

func HistoryShort(ctx context.Context, opts HistoryOptions) ([]*HistoryOutputShort, error) {
	wide, err := HistoryWide(ctx, opts)
	if err != nil {
		return nil, err
	}

	short := make([]*HistoryOutputShort, 0, len(wide))
	for _, w := range wide {
		s := &HistoryOutputShort{
			Name:    "-",
			EventID: "-",
			Type:    w.Type,
			Elapsed: w.Elapsed,
			Status:  w.Status,
			Details: "-",
		}

		if w.Name != nil {
			s.Name = *w.Name
		}

		if w.Details != nil {
			s.Details = *w.Details
		}
		if w.EventID != nil {
			s.EventID = fmt.Sprintf("%d", *w.EventID)
		}

		short = append(short, s)
	}

	return short, nil
}

func HistoryWide(ctx context.Context, opts HistoryOptions) ([]*HistoryOutputWide, error) {
	cli, err := dclient.DaprClient(ctx, dclient.Options{
		KubernetesMode: opts.KubernetesMode,
		Namespace:      opts.Namespace,
		AppID:          opts.AppID,
		RuntimePath:    runtime.GetDaprRuntimePath(),
	})
	if err != nil {
		return nil, err
	}
	defer cli.Cancel()

	history, err := cli.InstanceHistory(ctx, opts.InstanceID)
	if err != nil {
		return nil, err
	}

	var rows []*HistoryOutputWide
	var prevTs time.Time
	replay := 0

	for idx, ev := range history {
		ts := ev.GetTimestamp().AsTime()
		if idx == 0 {
			prevTs = ts
		}

		if _, ok := ev.GetEventType().(*protos.HistoryEvent_OrchestratorStarted); ok {
			replay++
		}

		row := &HistoryOutputWide{
			Namespace: opts.Namespace,
			AppID:     opts.AppID,
			Play:      replay,
			Type:      eventTypeName(ev),
			Name:      deriveName(ev),
			Timestamp: ts.Truncate(time.Second),
			Status:    deriveStatus(ev),
			Details:   deriveDetails(history[0], ev),
		}

		if idx == 0 {
			row.Elapsed = "Age:" + utils.HumanizeDuration(time.Since(ts))
		} else {
			row.Elapsed = utils.HumanizeDuration(ts.Sub(prevTs))
		}

		prevTs = ts

		if ev.EventId > 0 {
			row.EventID = ptr.Of(ev.EventId)
		}
		row.Router = routerStr(ev.Router)

		switch t := ev.GetEventType().(type) {
		case *protos.HistoryEvent_ExecutionStarted:
			if t.ExecutionStarted.OrchestrationInstance != nil &&
				t.ExecutionStarted.OrchestrationInstance.ExecutionId != nil {
				execID := t.ExecutionStarted.OrchestrationInstance.ExecutionId.Value
				row.ExecutionID = &execID
			}
			if t.ExecutionStarted.Input != nil {
				row.addAttr("input", trim(t.ExecutionStarted.Input, 120))
			}
			if len(t.ExecutionStarted.Tags) > 0 {
				row.addAttr("tags", flatTags(t.ExecutionStarted.Tags, 6))
			}
		case *protos.HistoryEvent_TaskScheduled:
			if row.EventID == nil {
				row.EventID = ptr.Of(int32(0))
			}
			if t.TaskScheduled.TaskExecutionId != "" {
				row.ExecutionID = ptr.Of(t.TaskScheduled.TaskExecutionId)
			}
			if t.TaskScheduled.Input != nil {
				row.addAttr("input", trim(t.TaskScheduled.Input, 120))
			}
		case *protos.HistoryEvent_TaskCompleted:
			row.addAttr("scheduledId", fmt.Sprintf("%d", t.TaskCompleted.TaskScheduledId))
			if t.TaskCompleted.TaskExecutionId != "" {
				row.ExecutionID = ptr.Of(t.TaskCompleted.TaskExecutionId)
			}
			if t.TaskCompleted.Result != nil {
				row.addAttr("result", trim(t.TaskCompleted.Result, 120))
			}
		case *protos.HistoryEvent_TaskFailed:
			row.addAttr("scheduledId", fmt.Sprintf("%d", t.TaskFailed.TaskScheduledId))
			if t.TaskFailed.TaskExecutionId != "" {
				row.ExecutionID = ptr.Of(t.TaskFailed.TaskExecutionId)
			}
			if fd := t.TaskFailed.FailureDetails; fd != nil {
				if fd.ErrorType != "" {
					row.addAttr("errorType", fd.ErrorType)
				}
				if fd.ErrorMessage != "" {
					row.addAttr("errorMsg", trim(wrapperspb.String(fd.ErrorMessage), 160))
				}
				if fd.IsNonRetriable {
					row.addAttr("nonRetriable", "true")
				}
			}
		case *protos.HistoryEvent_TimerCreated:
			if row.EventID == nil {
				row.EventID = ptr.Of(int32(0))
			}
			if t.TimerCreated.Name != nil {
				row.addAttr("timerName", *t.TimerCreated.Name)
			}
			row.addAttr("fireAt", t.TimerCreated.FireAt.AsTime().Format(time.RFC3339))
		case *protos.HistoryEvent_TimerFired:
			row.addAttr("timerId", fmt.Sprintf("%d", t.TimerFired.TimerId))
			row.addAttr("fireAt", t.TimerFired.FireAt.AsTime().Format(time.RFC3339))
		case *protos.HistoryEvent_EventRaised:
			row.addAttr("eventName", t.EventRaised.Name)
			if t.EventRaised.Input != nil {
				row.addAttr("payload", trim(t.EventRaised.Input, 120))
			}
		case *protos.HistoryEvent_EventSent:
			row.addAttr("eventName", t.EventSent.Name)
			if t.EventSent.Input != nil {
				row.addAttr("payload", trim(t.EventSent.Input, 120))
			}
			row.addAttr("targetInstance", t.EventSent.InstanceId)
		case *protos.HistoryEvent_ExecutionCompleted:
			if t.ExecutionCompleted.Result != nil {
				row.addAttr("output", trim(t.ExecutionCompleted.Result, 160))
			}
			if fd := t.ExecutionCompleted.FailureDetails; fd != nil {
				if fd.ErrorType != "" {
					row.addAttr("failureType", fd.ErrorType)
				}
				if fd.ErrorMessage != "" {
					row.addAttr("failureMessage", trim(wrapperspb.String(fd.ErrorMessage), 160))
				}
			}
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func eventTypeName(h *protos.HistoryEvent) string {
	switch h.GetEventType().(type) {
	case *protos.HistoryEvent_ExecutionStarted:
		return "ExecutionStarted"
	case *protos.HistoryEvent_ExecutionCompleted:
		return "ExecutionCompleted"
	case *protos.HistoryEvent_ExecutionTerminated:
		return "ExecutionTerminated"
	case *protos.HistoryEvent_TaskScheduled:
		return "TaskScheduled"
	case *protos.HistoryEvent_TaskCompleted:
		return "TaskCompleted"
	case *protos.HistoryEvent_TaskFailed:
		return "TaskFailed"
	case *protos.HistoryEvent_SubOrchestrationInstanceCreated:
		return "SubOrchCreated"
	case *protos.HistoryEvent_SubOrchestrationInstanceCompleted:
		return "SubOrchCompleted"
	case *protos.HistoryEvent_SubOrchestrationInstanceFailed:
		return "SubOrchFailed"
	case *protos.HistoryEvent_TimerCreated:
		return "TimerCreated"
	case *protos.HistoryEvent_TimerFired:
		return "TimerFired"
	case *protos.HistoryEvent_OrchestratorStarted:
		return "OrchestratorStarted"
	case *protos.HistoryEvent_OrchestratorCompleted:
		return "OrchestratorCompleted"
	case *protos.HistoryEvent_EventSent:
		return "EventSent"
	case *protos.HistoryEvent_EventRaised:
		return "EventRaised"
	case *protos.HistoryEvent_GenericEvent:
		return "GenericEvent"
	case *protos.HistoryEvent_HistoryState:
		return "HistoryState"
	case *protos.HistoryEvent_ContinueAsNew:
		return "ContinueAsNew"
	case *protos.HistoryEvent_ExecutionSuspended:
		return "ExecutionSuspended"
	case *protos.HistoryEvent_ExecutionResumed:
		return "ExecutionResumed"
	case *protos.HistoryEvent_EntityOperationSignaled:
		return "EntitySignaled"
	case *protos.HistoryEvent_EntityOperationCalled:
		return "EntityCalled"
	case *protos.HistoryEvent_EntityOperationCompleted:
		return "EntityCompleted"
	case *protos.HistoryEvent_EntityOperationFailed:
		return "EntityFailed"
	case *protos.HistoryEvent_EntityLockRequested:
		return "EntityLockRequested"
	case *protos.HistoryEvent_EntityLockGranted:
		return "EntityLockGranted"
	case *protos.HistoryEvent_EntityUnlockSent:
		return "EntityUnlockSent"
	default:
		return "Unknown"
	}
}

func deriveName(h *protos.HistoryEvent) *string {
	switch t := h.GetEventType().(type) {
	case *protos.HistoryEvent_TaskScheduled:
		return ptr.Of(t.TaskScheduled.Name)
	case *protos.HistoryEvent_TaskCompleted:
		return nil
	case *protos.HistoryEvent_TaskFailed:
		return nil
	case *protos.HistoryEvent_SubOrchestrationInstanceCreated:
		return ptr.Of(t.SubOrchestrationInstanceCreated.Name)
	case *protos.HistoryEvent_TimerCreated:
		if t.TimerCreated.Name != nil {
			return ptr.Of(*t.TimerCreated.Name)
		}
	case *protos.HistoryEvent_EventRaised:
		return ptr.Of(t.EventRaised.Name)
	case *protos.HistoryEvent_EventSent:
		return ptr.Of(t.EventSent.Name)
	case *protos.HistoryEvent_ExecutionStarted:
		return ptr.Of(t.ExecutionStarted.Name)
	}
	return nil
}

func deriveStatus(h *protos.HistoryEvent) string {
	switch t := h.GetEventType().(type) {
	case *protos.HistoryEvent_TaskFailed:
		return "FAILED"
	case *protos.HistoryEvent_ExecutionCompleted:
		return (workflow.WorkflowMetadata{RuntimeStatus: t.ExecutionCompleted.OrchestrationStatus}).String()
	case *protos.HistoryEvent_ExecutionTerminated:
		return "TERMINATED"
	case *protos.HistoryEvent_ExecutionSuspended:
		return "SUSPENDED"
	case *protos.HistoryEvent_ExecutionResumed:
		return "RESUMED"
	default:
		return "RUNNING"
	}
}

func deriveDetails(first *protos.HistoryEvent, h *protos.HistoryEvent) *string {
	switch t := h.GetEventType().(type) {
	case *protos.HistoryEvent_TaskScheduled:
		if in := t.TaskScheduled.RerunParentInstanceInfo; in != nil {
			return ptr.Of(fmt.Sprintf("rerun-parent=%s", in.InstanceID))
		}
		return nil
	case *protos.HistoryEvent_TimerCreated:
		det := fmt.Sprintf("fireAt=%s", t.TimerCreated.FireAt.AsTime().Format(time.RFC3339))
		if in := t.TimerCreated.RerunParentInstanceInfo; in != nil {
			det += fmt.Sprintf(",rerun-parent=%s", in.InstanceID)
		}
		return ptr.Of(det)
	case *protos.HistoryEvent_EventRaised:
		return ptr.Of(fmt.Sprintf("event=%s", t.EventRaised.Name))
	case *protos.HistoryEvent_EventSent:
		return ptr.Of(fmt.Sprintf("event=%s->%s", t.EventSent.Name, t.EventSent.InstanceId))
	case *protos.HistoryEvent_ExecutionStarted:
		return ptr.Of("workflow_start")
	case *protos.HistoryEvent_OrchestratorStarted:
		return ptr.Of("replay")
	case *protos.HistoryEvent_TaskCompleted:
		return ptr.Of(fmt.Sprintf("eventId=%d", t.TaskCompleted.TaskScheduledId))
	case *protos.HistoryEvent_ExecutionCompleted:
		return ptr.Of(fmt.Sprintf("execDuration=%s", utils.HumanizeDuration(h.GetTimestamp().AsTime().Sub(first.GetTimestamp().AsTime()))))
	case *protos.HistoryEvent_SubOrchestrationInstanceCreated:
		if in := t.SubOrchestrationInstanceCreated.RerunParentInstanceInfo; in != nil {
			return ptr.Of(fmt.Sprintf("rerun-parent=%s", in.InstanceID))
		}
		return nil
	default:
		return nil
	}
}

func routerStr(rt *protos.TaskRouter) *string {
	if rt == nil {
		return nil
	}
	if rt.TargetAppID != nil {
		return ptr.Of(fmt.Sprintf("%s->%s", rt.SourceAppID, *rt.TargetAppID))
	}
	return ptr.Of(rt.SourceAppID)
}

func (h *HistoryOutputWide) addAttr(key, val string) {
	if val == "" {
		return
	}
	if h.Attrs == nil {
		h.Attrs = ptr.Of(key + "=" + val)
		return
	}
	*h.Attrs += ";" + key + "=" + val
}

func flatTags(tags map[string]string, max int) string {
	i := 0
	var parts []string
	for k, v := range tags {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		i++
		if i >= max {
			break
		}
	}
	sort.Strings(parts)
	s := strings.Join(parts, ",")
	if len(tags) > max {
		s += ",…"
	}
	return s
}

func trim(ww *wrapperspb.StringValue, limit int) string {
	if ww == nil {
		return ""
	}

	s, err := dclient.UnquoteJSON([]byte(ww.Value))
	if err != nil {
		s = ww.Value
	}

	if limit <= 0 || len(s) <= limit {
		return s
	}

	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit]) + "…"
}
