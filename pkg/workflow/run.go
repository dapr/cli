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
	"time"

	"github.com/dapr/cli/pkg/workflow/dclient"
	"github.com/dapr/durabletask-go/workflow"
)

type RunOptions struct {
	KubernetesMode bool
	Namespace      string
	AppID          string
	Name           string
	InstanceID     *string
	Input          *string
	StartTime      *time.Time
}

func Run(ctx context.Context, opts RunOptions) (string, error) {
	cli, err := dclient.DaprClient(ctx, opts.KubernetesMode, opts.Namespace, opts.AppID)
	if err != nil {
		return "", err
	}
	defer cli.Cancel()

	wf := workflow.NewClient(cli.Dapr.GrpcClientConn())

	var wopts []workflow.NewWorkflowOptions
	if opts.InstanceID != nil {
		wopts = append(wopts, workflow.WithInstanceID(*opts.InstanceID))
	}
	if opts.Input != nil {
		wopts = append(wopts, workflow.WithInput(*opts.Input))
	}
	if opts.StartTime != nil {
		wopts = append(wopts, workflow.WithStartTime(*opts.StartTime))
	}

	return wf.ScheduleWorkflow(ctx, opts.Name, wopts...)
}
