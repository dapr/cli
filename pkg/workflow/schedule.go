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
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/dapr/cli/cmd/runtime"
	"github.com/dapr/cli/pkg/workflow/dclient"
	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/durabletask-go/workflow"
)

const (
	scheduleTimeFormat   = "Mon 02 Jan 15:04"
	scheduleResultLength = 80
)

type ScheduleOptions struct {
	KubernetesMode bool
	Namespace      string
	AppID          string
}

type CreateScheduleOptions struct {
	ScheduleOptions
	Name               string
	Workflow           string
	Schedule           string
	Input              *string
	DueTime            *time.Time
	ExpirationTime     *time.Time
	Repeats            *uint32
	TargetAppID        *string
	TargetAppNamespace *string
	Overwrite          bool
	SkipMissed         bool
}

type ScheduleOutputShort struct {
	Name   string `csv:"NAME"   json:"name"   yaml:"name"`
	Next   string `csv:"NEXT"   json:"next"   yaml:"next"`
	Last   string `csv:"LAST"   json:"last"   yaml:"last"`
	Status string `csv:"STATUS" json:"status" yaml:"status"`
}

type ScheduleOutputWide struct {
	Namespace      string     `csv:"-"        json:"namespace"                yaml:"namespace"`
	AppID          string     `csv:"-"        json:"appID"                    yaml:"appID"`
	Name           string     `csv:"NAME"     json:"name"                     yaml:"name"`
	Schedule       string     `csv:"SCHEDULE" json:"schedule"                 yaml:"schedule"`
	Runs           string     `csv:"RUNS"     json:"runs"                     yaml:"runs"`
	Next           string     `csv:"NEXT"     json:"-"                        yaml:"-"`
	Last           string     `csv:"LAST"     json:"-"                        yaml:"-"`
	Status         string     `csv:"STATUS"   json:"status"                   yaml:"status"`
	Result         string     `csv:"RESULT"   json:"result,omitempty"         yaml:"result,omitempty"`
	Workflow       string     `csv:"-"        json:"workflow"                 yaml:"workflow"`
	Input          string     `csv:"-"        json:"input,omitempty"          yaml:"input,omitempty"`
	State          string     `csv:"-"        json:"state"                    yaml:"state"`
	Created        time.Time  `csv:"-"        json:"created"                  yaml:"created"`
	NextRunTime    *time.Time `csv:"-"        json:"nextRunTime,omitempty"    yaml:"nextRunTime,omitempty"`
	LastRunTime    *time.Time `csv:"-"        json:"lastRunTime,omitempty"    yaml:"lastRunTime,omitempty"`
	RunCount       uint32     `csv:"-"        json:"runCount"                 yaml:"runCount"`
	SkipMissed     bool       `csv:"-"        json:"skipMissed,omitempty"     yaml:"skipMissed,omitempty"`
	LastInstanceID string     `csv:"-"        json:"lastInstanceID,omitempty" yaml:"lastInstanceID,omitempty"`
}

func CreateSchedule(ctx context.Context, opts CreateScheduleOptions) error {
	return withScheduleClient(ctx, opts.ScheduleOptions, func(wf *workflow.Client) error {
		_, err := wf.CreateWorkflowSchedule(ctx, opts.Name, opts.Workflow, opts.Schedule, createScheduleOptions(opts)...)
		return err
	})
}

func createScheduleOptions(opts CreateScheduleOptions) []workflow.CreateWorkflowScheduleOptions {
	var sopts []workflow.CreateWorkflowScheduleOptions
	if opts.Input != nil {
		sopts = append(sopts, workflow.WithRawScheduleInput(wrapperspb.String(jsonInput(*opts.Input))))
	}
	if opts.DueTime != nil {
		sopts = append(sopts, workflow.WithScheduleDueTime(*opts.DueTime))
	}
	if opts.ExpirationTime != nil {
		sopts = append(sopts, workflow.WithScheduleExpirationTime(*opts.ExpirationTime))
	}
	if opts.Repeats != nil {
		sopts = append(sopts, workflow.WithScheduleRepeats(*opts.Repeats))
	}
	if opts.TargetAppID != nil {
		sopts = append(sopts, workflow.WithScheduleTargetAppID(*opts.TargetAppID))
	}
	if opts.TargetAppNamespace != nil {
		sopts = append(sopts, workflow.WithScheduleTargetAppNamespace(*opts.TargetAppNamespace))
	}
	if opts.Overwrite {
		sopts = append(sopts, workflow.WithScheduleOverwrite())
	}
	if opts.SkipMissed {
		sopts = append(sopts, workflow.WithScheduleSkipMissed())
	}
	return sopts
}

func ListSchedules(ctx context.Context, opts ScheduleOptions) ([]*ScheduleOutputWide, error) {
	var out []*ScheduleOutputWide
	err := withScheduleClient(ctx, opts, func(wf *workflow.Client) error {
		var lopts []workflow.ListWorkflowSchedulesOptions
		for {
			resp, err := wf.ListWorkflowSchedules(ctx, lopts...)
			if err != nil {
				return err
			}
			for _, ws := range resp.GetSchedules() {
				out = append(out, scheduleOutput(opts, ws))
			}
			if resp.ContinuationToken == nil {
				return nil
			}
			lopts = []workflow.ListWorkflowSchedulesOptions{workflow.WithListWorkflowSchedulesContinuationToken(resp.GetContinuationToken())}
		}
	})
	return out, err
}

func GetSchedule(ctx context.Context, opts ScheduleOptions, name string) (*ScheduleOutputWide, error) {
	var out *ScheduleOutputWide
	err := withScheduleClient(ctx, opts, func(wf *workflow.Client) error {
		resp, err := wf.GetWorkflowSchedule(ctx, name)
		if err != nil {
			return err
		}
		out = scheduleOutput(opts, resp.GetSchedule())
		return nil
	})
	return out, err
}

func DeleteSchedule(ctx context.Context, opts ScheduleOptions, name string) error {
	return withScheduleClient(ctx, opts, func(wf *workflow.Client) error {
		_, err := wf.DeleteWorkflowSchedule(ctx, name)
		return err
	})
}

func TriggerSchedule(ctx context.Context, opts ScheduleOptions, name string) (string, error) {
	var id string
	err := withScheduleClient(ctx, opts, func(wf *workflow.Client) error {
		resp, err := wf.TriggerWorkflowSchedule(ctx, name)
		id = resp.GetInstanceId()
		return err
	})
	return id, err
}

func ShortSchedules(wide []*ScheduleOutputWide) []*ScheduleOutputShort {
	short := make([]*ScheduleOutputShort, len(wide))
	for i, w := range wide {
		short[i] = &ScheduleOutputShort{Name: w.Name, Next: w.Next, Last: w.Last, Status: w.Status}
	}
	return short
}

func withScheduleClient(ctx context.Context, opts ScheduleOptions, fn func(*workflow.Client) error) error {
	cli, err := dclient.DaprClient(ctx, dclient.Options{
		KubernetesMode: opts.KubernetesMode,
		Namespace:      opts.Namespace,
		AppID:          opts.AppID,
		RuntimePath:    runtime.GetDaprRuntimePath(),
	})
	if err != nil {
		return err
	}
	defer cli.Cancel()

	err = fn(cli.WF)
	if status.Code(err) == codes.Unimplemented {
		return errors.New("the Dapr runtime of this app does not support workflow schedules; upgrade it to a version that does")
	}
	return err
}

func scheduleOutput(opts ScheduleOptions, ws *workflow.WorkflowSchedule) *ScheduleOutputWide {
	st := ws.GetStatus()
	out := &ScheduleOutputWide{
		Namespace:      opts.Namespace,
		AppID:          opts.AppID,
		Name:           ws.GetName(),
		Schedule:       ws.GetSchedule(),
		Runs:           scheduleRuns(ws),
		Next:           "-",
		Last:           "-",
		Status:         scheduleStatus(st),
		Result:         scheduleResult(st.GetLastRun()),
		Workflow:       ws.GetWorkflowName(),
		Input:          ws.GetInput().GetValue(),
		State:          strings.ToLower(strings.TrimPrefix(st.GetState().String(), "WORKFLOW_SCHEDULE_STATE_")),
		Created:        st.GetCreatedAt().AsTime(),
		RunCount:       st.GetRunCount(),
		SkipMissed:     ws.GetSkipMissed(),
		LastInstanceID: st.GetLastRun().GetInstanceId(),
	}
	if t := st.GetNextRunTime(); t != nil {
		next := t.AsTime()
		out.NextRunTime, out.Next = &next, next.Local().Format(scheduleTimeFormat)
	}
	if t := st.GetLastRunTime(); t != nil {
		last := t.AsTime()
		out.LastRunTime, out.Last = &last, last.Local().Format(scheduleTimeFormat)
	}
	return out
}

// scheduleRuns is what each run starts: [<app>/]<workflow>(<input>).
func scheduleRuns(ws *workflow.WorkflowSchedule) string {
	runs := ws.GetWorkflowName()
	if app := ws.GetTargetAppID(); app != "" {
		runs = app + "/" + runs
	}
	if in := ws.GetInput(); in != nil {
		runs += "(" + displayJSON(in.GetValue()) + ")"
	}
	return runs
}

// scheduleStatus is how the last run went, or how the schedule itself stands
// when that says more.
func scheduleStatus(st *workflow.WorkflowScheduleStatus) string {
	run := st.GetLastRun()
	read := run != nil && run.RuntimeStatus != nil
	switch {
	case st.GetState() == workflow.WorkflowScheduleStateFailed:
		return "failed: " + firstLine(st.GetFailureMessage().GetValue())
	case st.GetState() == workflow.WorkflowScheduleStateCompleted && !read:
		return "completed"
	case !read:
		return "-"
	}
	took := run.GetLastUpdatedTimestamp().AsTime().Sub(run.GetCreatedTimestamp().AsTime())
	if took < time.Second {
		took = took.Round(time.Millisecond)
	} else {
		took = took.Round(time.Second)
	}
	switch run.GetRuntimeStatus() {
	case protos.OrchestrationStatus_ORCHESTRATION_STATUS_RUNNING,
		protos.OrchestrationStatus_ORCHESTRATION_STATUS_PENDING:
		return "running"
	case protos.OrchestrationStatus_ORCHESTRATION_STATUS_COMPLETED:
		return "ok " + took.String()
	case protos.OrchestrationStatus_ORCHESTRATION_STATUS_FAILED:
		return "failed " + took.String()
	default:
		return strings.ToLower(strings.TrimPrefix(run.GetRuntimeStatus().String(), "ORCHESTRATION_STATUS_"))
	}
}

// scheduleResult is the first line of the last run's failure or output.
func scheduleResult(run *workflow.WorkflowScheduleRun) string {
	res := run.GetFailureDetails().GetErrorMessage()
	if res == "" {
		res = displayJSON(run.GetOutput().GetValue())
	}
	res = firstLine(res)
	if r := []rune(res); len(r) > scheduleResultLength {
		res = string(r[:scheduleResultLength-1]) + "…"
	}
	return res
}

// jsonInput passes JSON input as is, and anything else as a JSON string.
func jsonInput(in string) string {
	if json.Valid([]byte(in)) {
		return in
	}
	b, _ := json.Marshal(in)
	return string(b)
}

// displayJSON shows a JSON string without its quotes.
func displayJSON(v string) string {
	var s string
	if json.Unmarshal([]byte(v), &s) == nil {
		return s
	}
	return v
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
