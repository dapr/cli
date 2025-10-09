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
	"os"
	"time"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/workflow/dclient"
	"github.com/dapr/durabletask-go/workflow"
)

type PurgeOptions struct {
	KubernetesMode bool
	Namespace      string
	AppID          string
	InstanceIDs    []string
	AllOlderThan   *time.Time
	All            bool

	ConnectionString *string
	TableName        *string
}

func Purge(ctx context.Context, opts PurgeOptions) error {
	cli, err := dclient.DaprClient(ctx, opts.KubernetesMode, opts.Namespace, opts.AppID)
	if err != nil {
		return err
	}
	defer cli.Cancel()

	var toPurge []string

	if len(opts.InstanceIDs) > 0 {
		toPurge = opts.InstanceIDs
	} else {
		var list []*ListOutputWide
		list, err = ListWide(ctx, ListOptions{
			KubernetesMode:   opts.KubernetesMode,
			Namespace:        opts.Namespace,
			AppID:            opts.AppID,
			ConnectionString: opts.ConnectionString,
			TableName:        opts.TableName,
			Filter: Filter{
				Terminal: true,
			},
		})
		if err != nil {
			return err
		}

		switch {
		case opts.AllOlderThan != nil:
			for _, w := range list {
				if w.Created.Before(*opts.AllOlderThan) {
					toPurge = append(toPurge, w.InstanceID)
				}
			}

		case opts.All:
			for _, w := range list {
				toPurge = append(toPurge, w.InstanceID)
			}
		}
	}

	wf := workflow.NewClient(cli.Dapr.GrpcClientConn())

	print.InfoStatusEvent(os.Stdout, "Purging %d workflow instance(s)", len(toPurge))

	for _, id := range toPurge {
		if err = wf.PurgeWorkflowState(ctx, id); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		print.SuccessStatusEvent(os.Stdout, "Purged workflow instance %q", id)
	}

	return nil
}
