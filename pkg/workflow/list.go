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
	"sort"
	"strings"
	"time"

	"github.com/dapr/cli/pkg/workflow/dclient"
	"github.com/dapr/durabletask-go/api"
	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/durabletask-go/workflow"
	"github.com/dapr/go-sdk/client"
	"github.com/dapr/kit/ptr"
	"k8s.io/apimachinery/pkg/util/duration"
)

type ListOptions struct {
	KubernetesMode   bool
	Namespace        string
	AppID            string
	ConnectionString *string
	TableName        *string
	Filter           Filter
}

type Filter struct {
	Name     *string
	Status   *string
	MaxAge   *time.Time
	Terminal bool
}

type ListOutputShort struct {
	Namespace     string `csv:"-" json:"namespace" yaml:"namespace"`
	AppID         string `csv:"-"    json:"appID"     yaml:"appID"`
	Name          string `csv:"NAME"    json:"name"     yaml:"name"`
	InstanceID    string `csv:"ID"    json:"instanceID"     yaml:"instanceID"`
	RuntimeStatus string `csv:"STATUS"    json:"runtimeStatus"     yaml:"runtimeStatus"`
	CustomStatus  string `csv:"CUSTOM STATUS"    json:"customStatus"     yaml:"customStatus"`
	Age           string `csv:"AGE"    json:"age"     yaml:"age"`
}

type ListOutputWide struct {
	Namespace      string    `csv:"NAMESPACE" json:"namespace" yaml:"namespace"`
	AppID          string    `csv:"APP ID"    json:"appID"     yaml:"appID"`
	Name           string    `csv:"Name"    json:"name"     yaml:"name"`
	InstanceID     string    `csv:"INSTANCE ID"    json:"instanceID"     yaml:"instanceID"`
	Created        time.Time `csv:"CREATED"    json:"created"     yaml:"created"`
	LastUpdate     time.Time `csv:"LAST UPDATE"    json:"lastUpdate"     yaml:"lastUpdate"`
	RuntimeStatus  string    `csv:"STATUS"    json:"runtimeStatus"     yaml:"runtimeStatus"`
	CustomStatus   string    `csv:"CUSTOM STATUS"    json:"customStatus"     yaml:"customStatus"`
	FailureMessage string    `csv:"FAILURE MESSAGE" json:"failureMessage"     yaml:"failureMessage"`
}

func ListShort(ctx context.Context, opts ListOptions) ([]*ListOutputShort, error) {
	wide, err := ListWide(ctx, opts)
	if err != nil {
		return nil, err
	}

	short := make([]*ListOutputShort, len(wide))
	for i, w := range wide {
		short[i] = &ListOutputShort{
			Namespace:     w.Namespace,
			AppID:         w.AppID,
			Name:          w.Name,
			InstanceID:    w.InstanceID,
			Age:           translateTimestampSince(w.Created),
			RuntimeStatus: w.RuntimeStatus,
		}
		if len(w.CustomStatus) > 0 {
			short[i].CustomStatus = w.CustomStatus
		}
	}

	return short, nil
}

func ListWide(ctx context.Context, opts ListOptions) ([]*ListOutputWide, error) {
	dclient, err := dclient.DaprClient(ctx, opts.KubernetesMode, opts.Namespace, opts.AppID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Dapr client: %w", err)
	}
	defer dclient.Cancel()

	connString := opts.ConnectionString
	if connString == nil {
		connString = dclient.ConnectionString
	}
	tableName := opts.TableName
	if tableName == nil {
		tableName = dclient.TableName
	}

	metaKeys, err := metakeys(ctx, DBOptions{
		Namespace:        opts.Namespace,
		AppID:            opts.AppID,
		Driver:           dclient.StateStoreDriver,
		ConnectionString: connString,
		TableName:        tableName,
	})
	if err != nil {
		return nil, err
	}

	return list(ctx, metaKeys, dclient.Dapr, opts)
}

func list(ctx context.Context, metaKeys []string, cl client.Client, opts ListOptions) ([]*ListOutputWide, error) {
	wf := workflow.NewClient(cl.GrpcClientConn())

	var listOutput []*ListOutputWide
	for _, key := range metaKeys {
		split := strings.Split(key, "||")
		if len(split) != 4 {
			continue
		}

		instanceID := split[2]

		resp, err := wf.FetchWorkflowMetadata(ctx, instanceID)
		if err != nil {
			return nil, err
		}

		if opts.Filter.Name != nil && resp.Name != *opts.Filter.Name {
			continue
		}
		if opts.Filter.Status != nil && resp.String() != *opts.Filter.Status {
			continue
		}
		if opts.Filter.MaxAge != nil && resp.CreatedAt.AsTime().Before(*opts.Filter.MaxAge) {
			continue
		}
		// TODO: @joshvanl: add `WorkflowIsCompleted` func to workflow package.
		if opts.Filter.Terminal && !api.OrchestrationMetadataIsComplete(ptr.Of(protos.OrchestrationMetadata(*resp))) {
			continue
		}

		wide := &ListOutputWide{
			Namespace:     opts.Namespace,
			AppID:         opts.AppID,
			Name:          resp.Name,
			InstanceID:    instanceID,
			Created:       resp.CreatedAt.AsTime().Truncate(time.Second),
			LastUpdate:    resp.LastUpdatedAt.AsTime().Truncate(time.Second),
			RuntimeStatus: resp.String(),
		}

		if resp.CustomStatus != nil {
			wide.CustomStatus = resp.CustomStatus.Value
		}

		if resp.FailureDetails != nil {
			wide.FailureMessage = strings.ReplaceAll(
				strings.ReplaceAll(
					resp.FailureDetails.GetErrorMessage(),
					"\n", ""),
				"\r", "")
		}

		listOutput = append(listOutput, wide)
	}

	sort.SliceStable(listOutput, func(i, j int) bool {
		if listOutput[i].Created.IsZero() {
			return false
		}
		if listOutput[j].Created.IsZero() {
			return true
		}
		return listOutput[i].Created.Before(listOutput[j].Created)
	})

	return listOutput, nil
}

func translateTimestampSince(timestamp time.Time) string {
	if timestamp.IsZero() {
		return "<scheduled>"
	}
	return duration.HumanDuration(time.Since(timestamp))
}
