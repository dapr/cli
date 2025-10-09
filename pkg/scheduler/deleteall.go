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

func DeleteAll(ctx context.Context, opts DeleteOptions, key string) error {
	etcdClient, cancel, err := EtcdClient(opts.KubernetesMode, opts.SchedulerNamespace)
	if err != nil {
		return err
	}
	defer cancel()

	split := strings.Split(key, "/")

	var paths []string
	switch split[0] {
	case "all":
		if len(split) != 1 {
			return fmt.Errorf("invalid key format: %s", key)
		}
		paths = []string{
			fmt.Sprintf("dapr/jobs/app||%s||", opts.DaprNamespace),
			fmt.Sprintf("dapr/jobs/actorreminder||%s||", opts.DaprNamespace),
			fmt.Sprintf("dapr/counters/app||%s||", opts.DaprNamespace),
			fmt.Sprintf("dapr/counters/actorreminder||%s||", opts.DaprNamespace),
		}
	case "app":
		switch len(split) {
		case 1:
			paths = []string{
				fmt.Sprintf("dapr/jobs/app||%s||", opts.DaprNamespace),
				fmt.Sprintf("dapr/counters/app||%s||", opts.DaprNamespace),
			}
		case 2:
			paths = []string{
				fmt.Sprintf("dapr/jobs/app||%s||%s||", opts.DaprNamespace, split[1]),
				fmt.Sprintf("dapr/counters/app||%s||%s||", opts.DaprNamespace, split[1]),
			}
		default:
			return fmt.Errorf("invalid key format: %s", key)
		}

	case "actor":
		switch len(split) {
		case 2:
			paths = []string{
				fmt.Sprintf("dapr/jobs/actorreminder||%s||%s||", opts.DaprNamespace, split[1]),
				fmt.Sprintf("dapr/counters/actorreminder||%s||%s||", opts.DaprNamespace, split[1]),
			}
		case 3:
			paths = []string{
				fmt.Sprintf("dapr/jobs/actorreminder||%s||%s||%s||", opts.DaprNamespace, split[1], split[2]),
				fmt.Sprintf("dapr/counters/actorreminder||%s||%s||%s||", opts.DaprNamespace, split[1], split[2]),
			}
		default:
			return fmt.Errorf("invalid key format: %s", key)
		}

	case "workflow":
		switch len(split) {
		case 1:
			paths = []string{
				fmt.Sprintf("dapr/jobs/actorreminder||%s||dapr.internal.%s.", opts.DaprNamespace, opts.DaprNamespace),
				fmt.Sprintf("dapr/counters/actorreminder||%s||dapr.internal.%s.", opts.DaprNamespace, opts.DaprNamespace),
			}
		case 2:
			paths = []string{
				fmt.Sprintf("dapr/jobs/actorreminder||%s||dapr.internal.%s.%s.workflow||", opts.DaprNamespace, opts.DaprNamespace, split[1]),
				fmt.Sprintf("dapr/jobs/actorreminder||%s||dapr.internal.%s.%s.activity||", opts.DaprNamespace, opts.DaprNamespace, split[1]),
				fmt.Sprintf("dapr/counters/actorreminder||%s||dapr.internal.%s.%s.workflow||", opts.DaprNamespace, opts.DaprNamespace, split[1]),
				fmt.Sprintf("dapr/counters/actorreminder||%s||dapr.internal.%s.%s.activity||", opts.DaprNamespace, opts.DaprNamespace, split[1]),
			}
		case 3:
			paths = []string{
				fmt.Sprintf("dapr/jobs/actorreminder||%s||dapr.internal.%s.%s.workflow||%s||", opts.DaprNamespace, opts.DaprNamespace, split[1], split[2]),
				fmt.Sprintf("dapr/jobs/actorreminder||%s||dapr.internal.%s.%s.activity||%s::", opts.DaprNamespace, opts.DaprNamespace, split[1], split[2]),
				fmt.Sprintf("dapr/counters/actorreminder||%s||dapr.internal.%s.%s.workflow||%s||", opts.DaprNamespace, opts.DaprNamespace, split[1], split[2]),
				fmt.Sprintf("dapr/counters/actorreminder||%s||dapr.internal.%s.%s.activity||%s::", opts.DaprNamespace, opts.DaprNamespace, split[1], split[2]),
			}
		default:
			return fmt.Errorf("invalid key format: %s", key)
		}

	default:
		return fmt.Errorf("unknown key prefix: %s", split[0])
	}

	oopts := make([]clientv3.Op, 0, len(paths))
	for _, path := range paths {
		oopts = append(oopts, clientv3.OpDelete(path,
			clientv3.WithPrefix(),
			clientv3.WithPrevKV(),
			clientv3.WithKeysOnly(),
		))
	}

	resp, err := etcdClient.Txn(ctx).Then(oopts...).Commit()
	if err != nil {
		return err
	}

	// Only count actual jobs, not counters.
	var deleted int64
	toCount := resp.Responses[:1]
	if len(paths) > 2 {
		toCount = resp.Responses[:2]
	}
	for _, resp := range toCount {
		for _, kv := range resp.GetResponseDeleteRange().GetPrevKvs() {
			print.InfoStatusEvent(os.Stdout, "Deleted job '%s'.", kv.Key)
		}
		deleted += resp.GetResponseDeleteRange().Deleted
	}

	print.InfoStatusEvent(os.Stdout, "Deleted %d jobs in namespace '%s'.", deleted, opts.DaprNamespace)

	return nil
}
