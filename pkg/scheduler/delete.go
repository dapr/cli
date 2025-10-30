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

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/dapr/cli/pkg/print"
)

type DeleteOptions struct {
	SchedulerNamespace string
	DaprNamespace      string
	KubernetesMode     bool
}

func Delete(ctx context.Context, opts DeleteOptions, keys ...string) error {
	etcdClient, cancel, err := EtcdClient(opts.KubernetesMode, opts.SchedulerNamespace)
	if err != nil {
		return err
	}
	defer cancel()

	for _, key := range keys {
		if err = delSingle(ctx, etcdClient, key, opts); err != nil {
			return err
		}

		print.InfoStatusEvent(os.Stdout, "Deleted %s in namespace '%s'.", key, opts.DaprNamespace)
	}

	return nil
}

func delSingle(ctx context.Context, client *clientv3.Client, key string, opts DeleteOptions) error {
	jobKey, err := parseJobKey(key)
	if err != nil {
		return err
	}

	paths := pathsFromJobKey(jobKey, opts.DaprNamespace)
	resp, err := client.Txn(ctx).Then(
		clientv3.OpDelete(paths[0]),
		clientv3.OpDelete(paths[1]),
	).Commit()
	if err != nil {
		return err
	}

	if len(resp.Responses) == 0 || resp.Responses[0].GetResponseDeleteRange().Deleted == 0 {
		return fmt.Errorf("no job with key '%s' found in namespace '%s'", key, opts.DaprNamespace)
	}

	return nil
}
