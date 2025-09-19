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
	schedulerv1 "github.com/dapr/dapr/pkg/proto/scheduler/v1"
	"github.com/dapr/kit/ptr"
)

const (
	FilterJobsAll   = "all"
	FilterJobsJob   = "jobs"
	FilterJobsActor = "actorreminder"
)

type ListJobsOptions struct {
	SchedulerNamespace string
	DaprNamespace      *string
	KubernetesMode     bool
	FilterJobType      string
}

type ListOutputWide struct {
	Namespace   string     `csv:"NAMESPACE" json:"namespace" yaml:"namespace"`
	AppID       string     `csv:"APP ID"    json:"appId"     yaml:"appId"`
	Name        string     `csv:"NAME" json:"name"  yaml:"name"`
	Target      string     `csv:"TARGET" json:"target"  yaml:"target"`
	Begin       time.Time  `csv:"BEGIN" json:"begin"  yaml:"begin,omitempty"`
	Expiration  *time.Time `csv:"EXPIRATION" json:"expiration"  yaml:"expiration,omitempty"`
	Schedule    *string    `csv:"SCHEDULE" json:"schedule"  yaml:"schedule,omitempty"`
	DueTime     *string    `csv:"DUE TIME" json:"dueTime"  yaml:"dueTime,omitempty"`
	TTL         *string    `csv:"TTL" json:"ttl"  yaml:"ttl,omitempty"`
	Repeats     *uint32    `csv:"REPEATS" json:"repeats"  yaml:"repeats,omitempty"`
	Count       uint32     `csv:"Count" json:"count"  yaml:"count,omitempty"`
	LastTrigger *time.Time `csv:"LAST TRIGGER" json:"lastTrigger"  yaml:"lastTrigger"`
}

type ListOutput struct {
	Namespace   string     `csv:"NAMESPACE" json:"namespace" yaml:"namespace"`
	AppID       string     `csv:"APP ID"    json:"appId"     yaml:"appId"`
	Name        string     `csv:"NAME" json:"name"  yaml:"name"`
	Target      string     `csv:"TARGET" json:"target"  yaml:"target"`
	Begin       time.Time  `csv:"BEGIN" json:"begin"  yaml:"begin,omitempty"`
	Count       uint32     `csv:"Count" json:"count"  yaml:"count,omitempty"`
	LastTrigger *time.Time `csv:"LAST TRIGGER" json:"lastTrigger"  yaml:"lastTrigger"`
}

type JobCount struct {
	Key     string
	Job     *stored.Job
	Counter *stored.Counter
}

func ListJobsAsOutput(ctx context.Context, opts ListJobsOptions) ([]ListOutput, error) {
	listWide, err := ListJobsAsOutputWide(ctx, opts)
	if err != nil {
		return nil, err
	}

	list := make([]ListOutput, 0, len(listWide))
	for _, item := range listWide {
		list = append(list, ListOutput{
			Namespace:   item.Namespace,
			AppID:       item.AppID,
			Name:        item.Name,
			Target:      item.Target,
			Begin:       item.Begin,
			Count:       item.Count,
			LastTrigger: item.LastTrigger,
		})
	}

	return list, nil
}

func ListJobsAsOutputWide(ctx context.Context, opts ListJobsOptions) ([]ListOutputWide, error) {
	jobCounters, err := ListJobs(ctx, opts)
	if err != nil {
		return nil, err
	}

	var list []ListOutputWide
	for _, jobCounter := range jobCounters {
		var meta schedulerv1.JobMetadata
		if err = jobCounter.Job.GetJob().GetMetadata().UnmarshalTo(&meta); err != nil {
			return nil, err
		}

		if opts.FilterJobType != FilterJobsAll {
			switch meta.GetTarget().GetType().(type) {
			case *schedulerv1.JobTargetMetadata_Job:
				if opts.FilterJobType != FilterJobsJob {
					continue
				}
			case *schedulerv1.JobTargetMetadata_Actor:
				if opts.FilterJobType != FilterJobsActor {
					continue
				}
			}
		}

		if opts.DaprNamespace != nil && meta.GetNamespace() != *opts.DaprNamespace {
			continue
		}

		listoutput := ListOutputWide{
			Name:      jobCounter.Key[(strings.LastIndex(jobCounter.Key, "||") + 2):],
			Namespace: meta.GetNamespace(),
			AppID:     meta.GetAppId(),
			Schedule:  jobCounter.Job.GetJob().Schedule,
			DueTime:   jobCounter.Job.GetJob().DueTime,
			TTL:       jobCounter.Job.GetJob().Ttl,
			Repeats:   jobCounter.Job.GetJob().Repeats,
		}

		switch meta.GetTarget().GetType().(type) {
		case *schedulerv1.JobTargetMetadata_Job:
			listoutput.Target = "job"
		case *schedulerv1.JobTargetMetadata_Actor:
			listoutput.Target = fmt.Sprintf("%s||%s)",
				meta.GetTarget().GetActor().GetType(),
				meta.GetTarget().GetActor().GetId(),
			)
		}

		switch t := jobCounter.Job.GetBegin().(type) {
		case *stored.Job_DueTime:
			listoutput.Begin = t.DueTime.AsTime().Truncate(time.Second)
		case *stored.Job_Start:
			listoutput.Begin = t.Start.AsTime().Truncate(time.Second)
		}

		if jobCounter.Job.Expiration != nil {
			listoutput.Expiration = ptr.Of(jobCounter.Job.Expiration.AsTime().Truncate(time.Second))
		}

		if jobCounter.Counter != nil {
			listoutput.Count = jobCounter.Counter.Count
			if jobCounter.Counter.LastTrigger != nil {
				listoutput.LastTrigger = ptr.Of(jobCounter.Counter.LastTrigger.AsTime().Truncate(time.Second))
			}
		}

		list = append(list, listoutput)
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Namespace < list[j].Namespace &&
			list[i].AppID < list[j].AppID &&
			list[i].Name < list[j].Name
	})

	return list, nil
}

func ListJobs(ctx context.Context, opts ListJobsOptions) ([]*JobCount, error) {
	etcdClient, cancel, err := etcdClient(opts.KubernetesMode, opts.SchedulerNamespace)
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

func listJobs(ctx context.Context, client *clientv3.Client) (map[string]*stored.Job, error) {
	//  dapr/jobs/actorreminder||default||dapr.internal.default.workflow-stress2.activity||e42d7040-e8e6-46d0-93be-d173ef8fd7d1-66::0::1||run-activity
	//  dapr/jobs/app||default||foobar

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
