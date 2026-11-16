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
	"errors"
	"fmt"

	"github.com/dapr/cli/cmd/runtime"
	"github.com/dapr/cli/pkg/workflow/dclient"
	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/durabletask-go/workflow"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type RaiseEventOptions struct {
	KubernetesMode bool
	Namespace      string
	AppID          string
	InstanceID     string
	Name           string
	Input          *string
}

func RaiseEvent(ctx context.Context, opts RaiseEventOptions) error {
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

	var wopts []workflow.RaiseEventOptions
	if opts.Input != nil {
		wopts = append(wopts, workflow.WithEventPayload(*opts.Input))
	}

	return cli.WF.RaiseEvent(ctx, opts.InstanceID, opts.Name, wopts...)
}

type SuspendOptions struct {
	KubernetesMode bool
	Namespace      string
	AppID          string
	InstanceID     string
	Reason         string
}

func Suspend(ctx context.Context, opts SuspendOptions) error {
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

	wf := workflow.NewClient(cli.Dapr.GrpcClientConn())

	return wf.SuspendWorkflow(ctx, opts.InstanceID, opts.Reason)
}

type ResumeOptions struct {
	KubernetesMode bool
	Namespace      string
	AppID          string
	InstanceID     string
	Reason         string
}

func Resume(ctx context.Context, opts ResumeOptions) error {
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

	wf := workflow.NewClient(cli.Dapr.GrpcClientConn())

	return wf.ResumeWorkflow(ctx, opts.InstanceID, opts.Reason)
}

type TerminateOptions struct {
	KubernetesMode bool
	Namespace      string
	AppID          string
	InstanceID     string
	Output         *string
	Reason         *string
}

func Terminate(ctx context.Context, opts TerminateOptions) error {
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

	wf := workflow.NewClient(cli.Dapr.GrpcClientConn())

	var wopts []workflow.TerminateOptions
	if opts.Output != nil {
		wopts = append(wopts, workflow.WithOutput(*opts.Output))
	}
	if opts.Reason != nil {
		wopts = append(wopts, withTerminateReason(*opts.Reason))
	}

	return wf.TerminateWorkflow(ctx, opts.InstanceID, wopts...)
}

const terminateRequestReasonField protowire.Number = 5

// withTerminateReason encodes the optional TerminateRequest.reason field while
// remaining compatible with durabletask-go versions whose generated request
// type does not yet know about field 5. Unknown protobuf fields are preserved
// by the durabletask gRPC client when it serializes the request.
func withTerminateReason(reason string) workflow.TerminateOptions {
	return func(req *protos.TerminateRequest) error {
		if req == nil {
			return errors.New("terminate request is nil")
		}

		value, err := proto.Marshal(wrapperspb.String(reason))
		if err != nil {
			return fmt.Errorf("failed to encode termination reason: %w", err)
		}

		unknown := req.ProtoReflect().GetUnknown()
		unknown = protowire.AppendTag(unknown, terminateRequestReasonField, protowire.BytesType)
		unknown = protowire.AppendBytes(unknown, value)
		req.ProtoReflect().SetUnknown(unknown)
		return nil
	}
}
