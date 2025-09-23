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
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/dapr/cli/pkg/workflow/dclient"
	"github.com/dapr/cli/utils"
	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/durabletask-go/workflow"
	"github.com/dapr/go-sdk/client"
	"github.com/dapr/kit/ptr"
)

const maxHistoryEntries = 1000

type HistoryOptions struct {
	KubernetesMode   bool
	Namespace        string
	AppID            string
	InstanceID       string
	ConnectionString *string
	SQLTableName     *string
}

type HistoryOutputWide struct {
	Namespace string    `csv:"-" json:"namespace,omitempty" yaml:"namespace,omitempty"`
	AppID     string    `csv:"-"    json:"appId"    yaml:"appId"`
	Play      int       `csv:"PLAY" json:"play" yaml:"play"`
	Type      string    `csv:"TYPE"      json:"type"      yaml:"type"` // e.g., TaskScheduled
	Name      *string   `csv:"NAME"      json:"name"      yaml:"name"` // activity/event/timer/orch
	EventID   *int32    `csv:"EVENTID,omitempty"      json:"eventId,omitempty"      yaml:"eventId,omitempty"`
	Timestamp time.Time `csv:"TIMESTAMP" json:"timestamp" yaml:"timestamp"`
	Elapsed   string    `csv:"ELAPSED" json:"elapsed" yaml:"elapsed"`    // "3.2ms" (empty if N/A)
	Status    string    `csv:"STATUS"    json:"status"    yaml:"status"` // "", "Failed", "Completed", ...
	// Short, single-line human summary that stays short on purpose.
	Details *string `csv:"DETAILS"   json:"details"   yaml:"details"` // e.g., "Activity=ChargeCard v1"
	// Light identifiers you might include in --wide
	Router      *string `csv:"ROUTER,omitempty"       json:"router,omitempty"       yaml:"router,omitempty"` // "src->tgt"
	ExecutionID *string `csv:"EXECUTION_ID,omitempty" json:"executionId,omitempty"  yaml:"executionId,omitempty"`

	// Everything else lives here and is shown only when requested.
	Attrs *string `csv:"ATTRS,omitempty" json:"attrs,omitempty" yaml:"attrs,omitempty"` // ordered key/vals
}

type HistoryOutputShort struct {
	Type    string `csv:"TYPE"      json:"type"      yaml:"type"` // e.g., TaskScheduled
	Name    string `csv:"NAME"      json:"name"      yaml:"name"` // activity/event/timer/orch
	EventID string `csv:"EVENTID,omitempty"      json:"eventId,omitempty"      yaml:"eventId,omitempty"`
	Elapsed string `csv:"ELAPSED" json:"elapsed" yaml:"elapsed"`    // "3.2ms" (empty if N/A)
	Status  string `csv:"STATUS"    json:"status"    yaml:"status"` // "", "Failed", "Completed", ...
	// Short, single-line human summary that stays short on purpose.
	Details string `csv:"DETAILS"   json:"details"   yaml:"details"` // e.g., "Activity=ChargeCard v1"
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
	cli, err := dclient.DaprClient(ctx, opts.KubernetesMode, opts.Namespace, opts.AppID)
	if err != nil {
		return nil, err
	}
	defer cli.Cancel()

	history, err := fetchHistory(ctx,
		cli.Dapr,
		"dapr.internal."+opts.Namespace+"."+opts.AppID+".workflow",
		opts.InstanceID,
	)
	if err != nil {
		return nil, err
	}

	// Sort: EventId if both present, else Timestamp
	sort.SliceStable(history, func(i, j int) bool {
		ei, ej := history[i], history[j]
		if ei.EventId > 0 && ej.EventId > 0 {
			return ei.EventId < ej.EventId
		}
		ti, tj := ei.GetTimestamp().AsTime(), ej.GetTimestamp().AsTime()
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return ei.EventId < ej.EventId
	})

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

func fetchHistory(ctx context.Context, cl client.Client, actorType, instanceID string) ([]*protos.HistoryEvent, error) {
	var events []*protos.HistoryEvent
	for startIndex := 0; startIndex <= 1; startIndex++ {
		if len(events) > 0 {
			break
		}

		for i := startIndex; i < maxHistoryEntries; i++ {
			key := fmt.Sprintf("history-%06d", i)

			resp, err := cl.GetActorState(ctx, &client.GetActorStateRequest{
				ActorType: actorType,
				ActorID:   instanceID,
				KeyName:   key,
			})
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					return nil, err
				}
				break
			}

			if resp == nil || len(resp.Data) == 0 {
				break
			}

			var event protos.HistoryEvent
			if err = decodeKey(resp.Data, &event); err != nil {
				return nil, fmt.Errorf("failed to decode history event %s: %w", key, err)
			}

			events = append(events, &event)
		}
	}

	return events, nil
}

func decodeKey(data []byte, item proto.Message) error {
	if len(data) == 0 {
		return fmt.Errorf("empty value")
	}

	if err := protojson.Unmarshal(data, item); err == nil {
		return nil
	}

	if unquoted, err := unquoteJSON(data); err == nil {
		if err := protojson.Unmarshal([]byte(unquoted), item); err == nil {
			return nil
		}
	}

	if err := proto.Unmarshal(data, item); err == nil {
		return nil
	}

	return fmt.Errorf("unable to decode history event (len=%d)", len(data))
}

func unquoteJSON(data []byte) (string, error) {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return "", err
	}
	return s, nil
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
		ver := ""
		if t.TaskScheduled.Version != nil && t.TaskScheduled.Version.Value != "" {
			ver = " v" + t.TaskScheduled.Version.Value
		}
		return ptr.Of(fmt.Sprintf("activity=%s%s", t.TaskScheduled.Name, ver))
	case *protos.HistoryEvent_TimerCreated:
		return ptr.Of(fmt.Sprintf("fireAt=%s", t.TimerCreated.FireAt.AsTime().Format(time.RFC3339)))
	case *protos.HistoryEvent_EventRaised:
		return ptr.Of(fmt.Sprintf("event=%s", t.EventRaised.Name))
	case *protos.HistoryEvent_EventSent:
		return ptr.Of(fmt.Sprintf("event=%s -> %s", t.EventSent.Name, t.EventSent.InstanceId))
	case *protos.HistoryEvent_ExecutionStarted:
		return ptr.Of("orchestration start")
	case *protos.HistoryEvent_OrchestratorStarted:
		return ptr.Of("replay cycle start")
	case *protos.HistoryEvent_TaskCompleted:
		return ptr.Of(fmt.Sprintf("eventId=%d", t.TaskCompleted.TaskScheduledId))
	case *protos.HistoryEvent_ExecutionCompleted:
		return ptr.Of(fmt.Sprintf("execDuration=%s", utils.HumanizeDuration(h.GetTimestamp().AsTime().Sub(first.GetTimestamp().AsTime()))))
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

	s, err := unquoteJSON([]byte(ww.Value))
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
