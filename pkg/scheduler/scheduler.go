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
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/scheduler/stored"
	schedulerv1 "github.com/dapr/dapr/pkg/proto/scheduler/v1"
	"github.com/dapr/kit/ptr"
)

const (
	FilterAll      = "all"
	FilterApp      = "app"
	FilterActor    = "actor"
	FilterWorkflow = "workflow"
	FilterActivity = "activity"
)

type Filter struct {
	Type      string
	Namespace *string
}

type jobKey struct {
	appID *string

	actorType *string
	actorID   *string

	instanceID *string
	activity   bool

	name string
}

func parseJob(jobCounter *JobCount, opts Filter) (*ListOutputWide, error) {
	var meta schedulerv1.JobMetadata
	if err := jobCounter.Job.GetJob().GetMetadata().UnmarshalTo(&meta); err != nil {
		return nil, err
	}

	if opts.Type != FilterAll {
		switch meta.GetTarget().GetType().(type) {
		case *schedulerv1.JobTargetMetadata_Job:
			if opts.Type != FilterApp {
				return nil, nil
			}
		case *schedulerv1.JobTargetMetadata_Actor:
			atype := meta.GetTarget().GetActor().GetType()
			switch {
			case strings.HasPrefix(atype, "dapr.internal.") && strings.HasSuffix(atype, ".workflow"):
				if opts.Type != FilterWorkflow {
					return nil, nil
				}
			case strings.HasPrefix(atype, "dapr.internal.") && strings.HasSuffix(atype, ".activity"):
				if opts.Type != FilterActivity {
					return nil, nil
				}
			default:
				if opts.Type != FilterActor {
					return nil, nil
				}
			}
		}
	}

	if opts.Namespace != nil && meta.GetNamespace() != *opts.Namespace {
		return nil, nil
	}

	listoutput := ListOutputWide{
		Name:      jobCounter.Key[(strings.LastIndex(jobCounter.Key, "||") + 2):],
		Namespace: meta.GetNamespace(),
		Schedule:  jobCounter.Job.GetJob().Schedule,
		DueTime:   jobCounter.Job.GetJob().DueTime,
		TTL:       jobCounter.Job.GetJob().Ttl,
		Repeats:   jobCounter.Job.GetJob().Repeats,
	}

	switch meta.GetTarget().GetType().(type) {
	case *schedulerv1.JobTargetMetadata_Job:
		listoutput.Name = "app/" + meta.GetAppId() + "/" + listoutput.Name
	case *schedulerv1.JobTargetMetadata_Actor:
		atype := meta.GetTarget().GetActor().GetType()
		switch {
		case strings.HasPrefix(atype, "dapr.internal.") && strings.HasSuffix(atype, ".workflow"):
			listoutput.Name = "workflow/" + fmt.Sprintf("%s/%s/%s",
				strings.Split(atype, ".")[3], meta.GetTarget().GetActor().GetId(),
				listoutput.Name,
			)
		case strings.HasPrefix(atype, "dapr.internal.") && strings.HasSuffix(atype, ".activity"):
			listoutput.Name = "activity/" + fmt.Sprintf("%s/%s",
				strings.Split(atype, ".")[3], meta.GetTarget().GetActor().GetId(),
			)
		default:
			listoutput.Name = "actor/" + fmt.Sprintf("%s/%s/%s",
				meta.GetTarget().GetActor().GetType(),
				meta.GetTarget().GetActor().GetId(),
				listoutput.Name,
			)
		}
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

	return &listoutput, nil
}

func parseJobKey(key string) (*jobKey, error) {
	split := strings.Split(key, "/")
	if len(split) < 2 {
		return nil, fmt.Errorf("failed to parse job key, expecting '{target type}/{identifier}', got '%s'", key)
	}

	switch split[0] {
	case FilterApp:
		if len(split) != 3 {
			return nil, fmt.Errorf("expecting job key to be in format 'app/{app ID}/{job name}', got '%s'", key)
		}
		return &jobKey{
			appID: &split[1],
			name:  split[2],
		}, nil

	case FilterActor:
		if len(split) != 4 {
			return nil, fmt.Errorf("expecting actor reminder key to be in format 'actor/{actor type}/{actor id}/{name}', got '%s'", key)
		}
		return &jobKey{
			actorType: &split[1],
			actorID:   &split[2],
			name:      split[3],
		}, nil

	case FilterWorkflow:
		if len(split) != 4 {
			return nil, fmt.Errorf("expecting workflow key to be in format 'workflow/{app ID}/{instance ID}/{name}', got '%s'", key)
		}
		return &jobKey{
			appID:      &split[1],
			instanceID: &split[2],
			name:       split[3],
		}, nil

	case FilterActivity:
		if len(split) != 3 {
			return nil, fmt.Errorf("expecting activity key to be in format 'activity/{app ID}/{activity ID}', got '%s'", key)
		}
		return &jobKey{
			appID:    &split[1],
			name:     split[2],
			activity: true,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported job type '%s', accepts 'app', 'actor', 'workflow', or 'activity'", split[0])
	}
}

func EtcdClient(kubernetesMode bool, schedulerNamespace string) (*clientv3.Client, context.CancelFunc, error) {
	var etcdClient *clientv3.Client
	var err error
	if kubernetesMode {
		var cancel context.CancelFunc
		etcdClient, cancel, err = etcdClientKubernetes(schedulerNamespace)
		if err != nil {
			return nil, nil, err
		}
		return etcdClient, cancel, nil
	} else {
		etcdClient, err = getEtcdClient("localhost:2379")
		if err != nil {
			return nil, nil, err
		}
	}

	return etcdClient, func() {}, nil
}

func getEtcdClient(host string) (*clientv3.Client, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{host},
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func etcdClientKubernetes(namespace string) (*clientv3.Client, context.CancelFunc, error) {
	config, _, err := kubernetes.GetKubeConfigClient()
	if err != nil {
		return nil, nil, err
	}

	portForward, err := kubernetes.NewPortForward(
		config,
		namespace,
		"dapr-scheduler-server-0",
		"localhost",
		2379,
		2379,
		false,
	)
	if err != nil {
		return nil, nil, err
	}

	if err = portForward.Init(); err != nil {
		return nil, nil, err
	}

	client, err := getEtcdClient("localhost:2379")
	if err != nil {
		return nil, nil, err
	}

	return client, portForward.Stop, nil
}
