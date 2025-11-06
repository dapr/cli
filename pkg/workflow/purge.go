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

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/dapr/cli/cmd/runtime"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/scheduler"
	"github.com/dapr/cli/pkg/workflow/dclient"
	"github.com/dapr/durabletask-go/workflow"
)

type PurgeOptions struct {
	KubernetesMode     bool
	Namespace          string
	SchedulerNamespace string
	AppID              string
	InstanceIDs        []string
	AllOlderThan       *time.Time
	All                bool

	ConnectionString *string
	TableName        *string
}

func Purge(ctx context.Context, opts PurgeOptions) error {
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

	etcdClient, cancel, err := scheduler.EtcdClient(opts.KubernetesMode, opts.SchedulerNamespace)
	if err != nil {
		return err
	}
	defer cancel()

	print.InfoStatusEvent(os.Stdout, "Purging %d workflow instance(s)", len(toPurge))

	for _, id := range toPurge {
		if err = wf.PurgeWorkflowState(ctx, id); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}

		paths := []string{
			fmt.Sprintf("dapr/jobs/actorreminder||%s||dapr.internal.%s.%s.workflow||%s||", opts.Namespace, opts.Namespace, opts.AppID, id),
			fmt.Sprintf("dapr/jobs/actorreminder||%s||dapr.internal.%s.%s.activity||%s::", opts.Namespace, opts.Namespace, opts.AppID, id),
			fmt.Sprintf("dapr/counters/actorreminder||%s||dapr.internal.%s.%s.workflow||%s||", opts.Namespace, opts.Namespace, opts.AppID, id),
			fmt.Sprintf("dapr/counters/actorreminder||%s||dapr.internal.%s.%s.activity||%s::", opts.Namespace, opts.Namespace, opts.AppID, id),
		}

		oopts := make([]clientv3.Op, 0, len(paths))
		for _, path := range paths {
			oopts = append(oopts, clientv3.OpDelete(path,
				clientv3.WithPrefix(),
				clientv3.WithPrevKV(),
				clientv3.WithKeysOnly(),
			))
		}

		if _, err = etcdClient.Txn(ctx).Then(oopts...).Commit(); err != nil {
			return err
		}

		print.SuccessStatusEvent(os.Stdout, "Purged workflow instance %q", id)
	}

	return nil
}
