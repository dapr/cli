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

package scheduler

import (
	"context"
	"fmt"
	"os"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/dapr/cli/pkg/print"
)

type DeleteOptions struct {
	SchedulerNamespace string
	DaprNamespace      string
	KubernetesMode     bool
}

func DeleteAll(ctx context.Context, opts DeleteOptions) error {
	etcdClient, cancel, err := etcdClient(opts.KubernetesMode, opts.SchedulerNamespace)
	if err != nil {
		return err
	}
	defer cancel()

	resp, err := etcdClient.Txn(ctx).Then(
		clientv3.OpDelete(fmt.Sprintf("dapr/jobs/app||%s||", opts.DaprNamespace), clientv3.WithPrefix()),
		clientv3.OpDelete(fmt.Sprintf("dapr/jobs/actorreminder||%s||", opts.DaprNamespace), clientv3.WithPrefix()),
		clientv3.OpDelete(fmt.Sprintf("dapr/counters/app||%s||", opts.DaprNamespace), clientv3.WithPrefix()),
		clientv3.OpDelete(fmt.Sprintf("dapr/counters/actorreminder||%s||", opts.DaprNamespace), clientv3.WithPrefix()),
	).Commit()
	if err != nil {
		return err
	}

	// Only count actual jobs, not counters.
	var deleted int64
	for _, resp := range resp.Responses[:2] {
		deleted += resp.GetResponseDeleteRange().Deleted
	}

	print.InfoStatusEvent(os.Stdout, "Deleted %d jobs in namespace '%s'.", deleted, opts.DaprNamespace)

	return nil
}

func Delete(ctx context.Context, key string, opts DeleteOptions) error {
	split := strings.Split(key, "/")
	if len(split) < 2 {
		return fmt.Errorf("failed to parse job key, expecting '{target type}/{identifier}', got '%s'", key)
	}

	etcdClient, cancel, err := etcdClient(opts.KubernetesMode, opts.SchedulerNamespace)
	if err != nil {
		return err
	}
	defer cancel()

	switch split[0] {
	case "job":
		if len(split) != 3 {
			return fmt.Errorf("expecting job key to be in format 'job/{app ID}/{job name}', got '%s'", key)
		}
		return deleteJob(ctx, etcdClient, split[1], split[2], opts)
	case "actorreminder":
		if len(split) != 2 {
			return fmt.Errorf("expecting actor reminder key to be in format 'actorreminder/{actor type}||{actor id}||{reminder name}', got '%s'", key)
		}
		actorSplit := strings.Split(split[1], "||")
		if len(actorSplit) != 3 {
			return fmt.Errorf(
				"failed to parse actor reminder key, expecting 'actorreminder/{actor type}||{actor id}||{reminder name}', got '%s'",
				key,
			)
		}

		return deleteActorReminder(ctx, etcdClient, actorSplit[0], actorSplit[1], actorSplit[2], opts)
	default:
		return fmt.Errorf("unsupported job target type '%s', accepts 'job' and 'actorreminder'", split[0])
	}

	return nil
}

func deleteJob(ctx context.Context,
	client *clientv3.Client,
	appID, name string,
	opts DeleteOptions,
) error {
	return deleteKeys(ctx,
		client,
		fmt.Sprintf("dapr/jobs/app||%s||%s||%s", opts.DaprNamespace, appID, name),
		fmt.Sprintf("dapr/counters/app||%s||%s||%s", opts.DaprNamespace, appID, name),
		opts,
	)
}

func deleteActorReminder(ctx context.Context,
	client *clientv3.Client,
	actorType, actorID, name string,
	opts DeleteOptions,
) error {
	return deleteKeys(ctx,
		client,
		fmt.Sprintf("dapr/jobs/actorreminder||%s||%s||%s||%s",
			opts.DaprNamespace, actorType, actorID, name,
		),
		fmt.Sprintf("dapr/counters/actorreminder||%s||%s||%s||%s",
			opts.DaprNamespace, actorType, actorID, name,
		),
		opts,
	)
}

func deleteKeys(ctx context.Context, client *clientv3.Client, key1, key2 string, opts DeleteOptions) error {
	resp, err := client.Txn(ctx).Then(
		clientv3.OpDelete(key1),
		clientv3.OpDelete(key2),
	).Commit()
	if err != nil {
		return err
	}

	if len(resp.Responses) == 0 || resp.Responses[0].GetResponseDeleteRange().Deleted == 0 {
		return fmt.Errorf("no job with key '%s' found in namespace '%s'", key1, opts.DaprNamespace)
	}

	return nil
}
