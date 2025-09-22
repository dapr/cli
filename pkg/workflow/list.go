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
	"sort"
	"strings"
	"time"

	"github.com/dapr/cli/pkg/workflow/dclient"
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
	SQLTableName     *string

	FilterWorkflowName   *string
	FilterWorkflowStatus *string
	FilterMaxAge         *time.Time
}

type ListOutputShort struct {
	Namespace     string `csv:"-" json:"namespace" yaml:"namespace"`
	AppID         string `csv:"-"    json:"appId"     yaml:"appId"`
	InstanceID    string `csv:"INSTANCE ID"    json:"instanceID"     yaml:"instanceID"`
	Name          string `csv:"NAME"    json:"name"     yaml:"name"`
	RuntimeStatus string `csv:"STATUS"    json:"runtimeStatus"     yaml:"runtimeStatus"`
	Age           string `csv:"AGE"    json:"age"     yaml:"age"`
}

type ListOutputWide struct {
	Namespace      string    `csv:"NAMESPACE" json:"namespace" yaml:"namespace"`
	AppID          string    `csv:"APP ID"    json:"appId"     yaml:"appId"`
	InstanceID     string    `csv:"INSTANCE ID"    json:"instanceID"     yaml:"instanceID"`
	Name           string    `csv:"Name"    json:"name"     yaml:"name"`
	Created        time.Time `csv:"CREATED"    json:"created"     yaml:"created"`
	LastUpdate     time.Time `csv:"LAST UPDATE"    json:"lastUpdate"     yaml:"lastUpdate"`
	RuntimeStatus  string    `csv:"STATUS"    json:"runtimeStatus"     yaml:"runtimeStatus"`
	CustomStatus   *string   `csv:"CUSTOM STATUS"    json:"customStatus"     yaml:"customStatus"`
	FailureMessage *string   `csv:"FAILURE MESSAGE" json:"failureMessage"     yaml:"failureMessage"`
	FailureType    *string   `csv:"FAILURE TYPE"    json:"failureType"     yaml:"failureType"`
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
	}

	return short, nil
}

func ListWide(ctx context.Context, opts ListOptions) ([]*ListOutputWide, error) {
	dclient, err := dclient.DaprClient(ctx, opts.KubernetesMode, opts.Namespace, opts.AppID)
	if err != nil {
		return nil, err
	}
	defer dclient.Cancel()

	connString := opts.ConnectionString
	if connString == nil {
		connString = dclient.ConnectionString
	}
	tableName := opts.SQLTableName
	if tableName == nil {
		tableName = dclient.SQLTableName
	}

	metaKeys, err := metakeys(ctx, DBOptions{
		Namespace:        opts.Namespace,
		AppID:            opts.AppID,
		Driver:           dclient.StateStoreDriver,
		ConnectionString: connString,
		SQLTableName:     tableName,
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

		if opts.FilterWorkflowName != nil && resp.Name != *opts.FilterWorkflowName {
			continue
		}
		if opts.FilterWorkflowStatus != nil && resp.String() != *opts.FilterWorkflowStatus {
			continue
		}
		if opts.FilterMaxAge != nil && resp.CreatedAt.AsTime().Before(*opts.FilterMaxAge) {
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
			wide.CustomStatus = ptr.Of(resp.CustomStatus.Value)
		}

		if resp.FailureDetails != nil {
			wide.FailureMessage = ptr.Of(resp.FailureDetails.GetErrorMessage())
			wide.FailureType = ptr.Of(resp.FailureDetails.GetErrorType())
		}

		listOutput = append(listOutput, wide)
	}

	sort.SliceStable(listOutput, func(i, j int) bool {
		return listOutput[i].Created.Before(listOutput[j].Created)
	})

	return listOutput, nil
}

func translateTimestampSince(timestamp time.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}
	return duration.HumanDuration(time.Since(timestamp))
}
