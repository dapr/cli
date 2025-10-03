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

	"github.com/dapr/cli/pkg/scheduler/stored"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"
)

type GetOptions struct {
	SchedulerNamespace string
	DaprNamespace      string
	KubernetesMode     bool
}

func Get(ctx context.Context, opts GetOptions, keys ...string) ([]*ListOutput, error) {
	list, err := GetWide(ctx, opts, keys...)
	if err != nil {
		return nil, err
	}

	return listWideToShort(list)
}

func GetWide(ctx context.Context, opts GetOptions, keys ...string) ([]*ListOutputWide, error) {
	etcdClient, cancel, err := etcdClient(opts.KubernetesMode, opts.SchedulerNamespace)
	if err != nil {
		return nil, err
	}
	defer cancel()

	results := make([]*ListOutputWide, 0, len(keys))
	for _, key := range keys {
		wide, err := getSingle(ctx, etcdClient, key, opts)
		if err != nil {
			return nil, err
		}

		results = append(results, wide)
	}

	return results, nil
}

func getSingle(ctx context.Context, cl *clientv3.Client, key string, opts GetOptions) (*ListOutputWide, error) {
	jobKey, err := parseJobKey(key)
	if err != nil {
		return nil, err
	}

	paths := pathsFromJobKey(jobKey, opts.DaprNamespace)

	resp, err := cl.Txn(ctx).Then(
		clientv3.OpGet(paths[0]),
		clientv3.OpGet(paths[1]),
	).Commit()
	if err != nil {
		return nil, err
	}

	if len(resp.Responses[0].GetResponseRange().Kvs) == 0 {
		return nil, fmt.Errorf("job '%s' not found", key)
	}

	var storedJ stored.Job
	if err = proto.Unmarshal(resp.Responses[0].GetResponseRange().Kvs[0].Value, &storedJ); err != nil {
		return nil, err
	}

	var storedC stored.Counter
	if kvs := resp.Responses[1].GetResponseRange().Kvs; len(kvs) > 0 {
		if err = proto.Unmarshal(kvs[0].Value, &storedC); err != nil {
			return nil, err
		}
	}

	return parseJob(&JobCount{
		Key:     paths[0],
		Job:     &storedJ,
		Counter: &storedC,
	}, Filter{
		Type: FilterAll,
	})
}

func pathsFromJobKey(jobKey *jobKey, namespace string) [2]string {
	var paths [2]string
	switch {
	case jobKey.actorType != nil:
		paths[0] = fmt.Sprintf("dapr/jobs/actorreminder||%s||%s||%s||%s",
			namespace, *jobKey.actorType, *jobKey.actorID, jobKey.name,
		)
		paths[1] = fmt.Sprintf("dapr/counters/actorreminder||%s||%s||%s||%s",
			namespace, *jobKey.actorType, *jobKey.actorID, jobKey.name,
		)

	case jobKey.activity:
		actorType := fmt.Sprintf("dapr.internal.%s.%s.activity", namespace, *jobKey.appID)
		actorID := jobKey.name
		paths[0] = fmt.Sprintf("dapr/jobs/actorreminder||%s||%s||%s||run-activity",
			namespace, actorType, actorID,
		)
		paths[1] = fmt.Sprintf("dapr/counters/actorreminder||%s||%s||%s||run-activity",
			namespace, actorType, actorID,
		)

	case jobKey.instanceID != nil:
		actorType := fmt.Sprintf("dapr.internal.%s.%s.workflow", namespace, *jobKey.appID)
		actorID := *jobKey.instanceID
		paths[0] = fmt.Sprintf("dapr/jobs/actorreminder||%s||%s||%s||%s",
			namespace, actorType, actorID, jobKey.name,
		)
		paths[1] = fmt.Sprintf("dapr/counters/actorreminder||%s||%s||%s||%s",
			namespace, actorType, actorID, jobKey.name,
		)

	default:
		paths[0] = fmt.Sprintf("dapr/jobs/app||%s||%s||%s", namespace, *jobKey.appID, jobKey.name)
		paths[1] = fmt.Sprintf("dapr/counters/app||%s||%s||%s", namespace, *jobKey.appID, jobKey.name)
	}

	return paths
}
