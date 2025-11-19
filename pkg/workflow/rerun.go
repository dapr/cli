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

	"github.com/dapr/cli/cmd/runtime"
	"github.com/dapr/cli/pkg/workflow/dclient"
	"github.com/dapr/durabletask-go/workflow"
)

type ReRunOptions struct {
	KubernetesMode bool
	Namespace      string
	AppID          string
	InstanceID     string
	EventID        uint32
	NewInstanceID  *string
	Input          *string
}

func ReRun(ctx context.Context, opts ReRunOptions) (string, error) {
	cli, err := dclient.DaprClient(ctx, dclient.Options{
		KubernetesMode: opts.KubernetesMode,
		Namespace:      opts.Namespace,
		AppID:          opts.AppID,
		RuntimePath:    runtime.GetDaprRuntimePath(),
	})
	if err != nil {
		return "", err
	}
	defer cli.Cancel()

	var wopts []workflow.RerunOptions
	if opts.NewInstanceID != nil {
		wopts = append(wopts, workflow.WithRerunNewInstanceID(*opts.NewInstanceID))
	}
	if opts.Input != nil {
		wopts = append(wopts, workflow.WithRerunInput(*opts.Input))
	}

	return cli.WF.RerunWorkflowFromEvent(ctx, opts.InstanceID, opts.EventID, wopts...)
}
