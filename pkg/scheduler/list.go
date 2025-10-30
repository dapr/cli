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
	"sort"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"

	"github.com/dapr/cli/pkg/scheduler/stored"
	"github.com/dapr/cli/utils"
)

type ListOptions struct {
	SchedulerNamespace string
	KubernetesMode     bool
	Filter             Filter
}

type ListOutputWide struct {
	Namespace   string     `csv:"NAMESPACE" json:"namespace" yaml:"namespace"`
	Name        string     `csv:"NAME" json:"name"  yaml:"name"`
	Begin       time.Time  `csv:"BEGIN" json:"begin"  yaml:"begin,omitempty"`
	Expiration  *time.Time `csv:"EXPIRATION" json:"expiration"  yaml:"expiration,omitempty"`
	Schedule    *string    `csv:"SCHEDULE" json:"schedule"  yaml:"schedule,omitempty"`
	DueTime     *string    `csv:"DUE TIME" json:"dueTime"  yaml:"dueTime,omitempty"`
	TTL         *string    `csv:"TTL" json:"ttl"  yaml:"ttl,omitempty"`
	Repeats     *uint32    `csv:"REPEATS" json:"repeats"  yaml:"repeats,omitempty"`
	Count       uint32     `csv:"COUNT" json:"count"  yaml:"count,omitempty"`
	LastTrigger *time.Time `csv:"LAST TRIGGER" json:"lastTrigger,omitempty"  yaml:"lastTrigger,omitempty"`
}

type ListOutput struct {
	Name        string `csv:"NAME" json:"name"  yaml:"name"`
	Begin       string `csv:"BEGIN" json:"begin"  yaml:"begin,omitempty"`
	Count       uint32 `csv:"COUNT" json:"count"  yaml:"count,omitempty"`
	LastTrigger string `csv:"LAST TRIGGER" json:"lastTrigger" yaml:"lastTrigger"`
}

type JobCount struct {
	Key     string
	Job     *stored.Job
	Counter *stored.Counter
}

func List(ctx context.Context, opts ListOptions) ([]*ListOutput, error) {
	listWide, err := ListWide(ctx, opts)
	if err != nil {
		return nil, err
	}

	return listWideToShort(listWide)
}

func ListWide(ctx context.Context, opts ListOptions) ([]*ListOutputWide, error) {
	jobCounters, err := ListJobs(ctx, opts)
	if err != nil {
		return nil, err
	}

	var list []*ListOutputWide
	for _, jobCounter := range jobCounters {
		listoutput, err := parseJob(jobCounter, opts.Filter)
		if err != nil {
			return nil, err
		}

		if listoutput == nil {
			continue
		}

		list = append(list, listoutput)
	}

	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Namespace == list[j].Namespace {
			if list[i].Begin.Equal(list[j].Begin) {
				return list[i].Name < list[j].Name
			}
			return list[i].Begin.Before(list[j].Begin)
		}
		return list[i].Namespace < list[j].Namespace
	})

	return list, nil
}

func ListJobs(ctx context.Context, opts ListOptions) ([]*JobCount, error) {
	etcdClient, cancel, err := EtcdClient(opts.KubernetesMode, opts.SchedulerNamespace)
	if err != nil {
		return nil, err
	}
	defer cancel()

	jobs, err := listJobs(ctx, etcdClient)
	if err != nil {
		return nil, err
	}

	counters, err := listCounters(ctx, etcdClient)
	if err != nil {
		return nil, err
	}

	jobCounts := make([]*JobCount, 0, len(jobs))
	for key, job := range jobs {
		jobCount := &JobCount{
			Key: key,
			Job: job,
		}

		counter, ok := counters[strings.ReplaceAll(key, "dapr/jobs/", "dapr/counters/")]
		if ok {
			jobCount.Counter = counter
		}

		jobCounts = append(jobCounts, jobCount)
	}

	return jobCounts, nil
}

func listWideToShort(listWide []*ListOutputWide) ([]*ListOutput, error) {
	now := time.Now()
	list := make([]*ListOutput, 0, len(listWide))
	for _, item := range listWide {
		if item == nil {
			continue
		}

		l := ListOutput{
			Name:  item.Name,
			Count: item.Count,
		}

		if item.LastTrigger != nil {
			l.LastTrigger = "-" + utils.HumanizeDuration(now.Sub(*item.LastTrigger))
		}

		if item.Begin.After(now) {
			l.Begin = "+" + utils.HumanizeDuration(item.Begin.Sub(now))
		} else {
			l.Begin = "-" + utils.HumanizeDuration(now.Sub(item.Begin))
		}

		list = append(list, &l)
	}

	return list, nil
}

func listJobs(ctx context.Context, client *clientv3.Client) (map[string]*stored.Job, error) {
	resp, err := client.Get(ctx,
		"dapr/jobs/",
		clientv3.WithPrefix(),
		clientv3.WithLimit(0),
	)
	if err != nil {
		return nil, err
	}

	jobs := make(map[string]*stored.Job)
	for _, kv := range resp.Kvs {
		var stored stored.Job
		if err := proto.Unmarshal(kv.Value, &stored); err != nil {
			return nil, fmt.Errorf("failed to unmarshal job %s: %w", kv.Key, err)
		}

		jobs[string(kv.Key)] = &stored
	}

	return jobs, nil
}

func listCounters(ctx context.Context, client *clientv3.Client) (map[string]*stored.Counter, error) {
	resp, err := client.Get(ctx,
		"dapr/counters/",
		clientv3.WithPrefix(),
		clientv3.WithLimit(0),
	)
	if err != nil {
		return nil, err
	}

	counters := make(map[string]*stored.Counter)
	for _, kv := range resp.Kvs {
		var stored stored.Counter
		if err := proto.Unmarshal(kv.Value, &stored); err != nil {
			return nil, fmt.Errorf("failed to unmarshal counter %s: %w", kv.Key, err)
		}

		counters[string(kv.Key)] = &stored
	}

	return counters, nil
}
