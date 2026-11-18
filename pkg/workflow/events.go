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

	"github.com/dapr/cli/cmd/runtime"
	"github.com/dapr/cli/pkg/workflow/dclient"
	"github.com/dapr/durabletask-go/workflow"
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

	wopts, err := terminateWorkflowOptions(opts.Output, opts.Reason)
	if err != nil {
		return err
	}

	return cli.WF.TerminateWorkflow(ctx, opts.InstanceID, wopts...)
}

// terminateWorkflowOptions maps the CLI reason onto the existing workflow
// termination input. The Durable Task protocol has one termination payload,
// so output and reason cannot be supplied independently.
func terminateWorkflowOptions(output, reason *string) ([]workflow.TerminateOptions, error) {
	if output != nil && reason != nil {
		return nil, errors.New("--output and --reason cannot be used together")
	}

	switch {
	case reason != nil:
		return []workflow.TerminateOptions{workflow.WithOutput(*reason)}, nil
	case output != nil:
		return []workflow.TerminateOptions{workflow.WithOutput(*output)}, nil
	default:
		return nil, nil
	}
}
