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
	"strings"
	"time"

	"github.com/dapr/cli/pkg/workflow/dclient"
	"github.com/dapr/durabletask-go/workflow"
	"github.com/dapr/go-sdk/client"
	"github.com/dapr/kit/ptr"
)

const maxHistoryEntries = 1000

type ListOptions struct {
	KubernetesMode   bool
	Namespace        string
	AppID            string
	ConnectionString *string
	SQLTableName     *string

	FilterWorkflowName   *string
	FilterWorkflowStatus *string
}

type ListOutputShort struct {
	Namespace     string    `csv:"NAMESPACE" json:"namespace" yaml:"namespace"`
	AppID         string    `csv:"APP ID"    json:"appId"     yaml:"appId"`
	Workflow      string    `csv:"WORKFLOW"    json:"workflow"     yaml:"workflow"`
	InstanceID    string    `csv:"INSTANCE ID"    json:"instanceID"     yaml:"instanceID"`
	Created       time.Time `csv:"CREATED"    json:"created"     yaml:"created"`
	RuntimeStatus string    `csv:"STATUS"    json:"runtimeStatus"     yaml:"runtimeStatus"`
}

type ListOutputWide struct {
	Namespace      string    `csv:"NAMESPACE" json:"namespace" yaml:"namespace"`
	AppID          string    `csv:"APP ID"    json:"appId"     yaml:"appId"`
	Workflow       string    `csv:"WORKFLOW"    json:"workflow"     yaml:"workflow"`
	InstanceID     string    `csv:"INSTANCE ID"    json:"instanceID"     yaml:"instanceID"`
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
			Workflow:      w.Workflow,
			InstanceID:    w.InstanceID,
			Created:       w.Created,
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

	metaKeys, err := metakeys(ctx, DBOptions{
		Namespace:        opts.Namespace,
		AppID:            opts.AppID,
		Driver:           dclient.StateStoreDriver,
		ConnectionString: opts.ConnectionString,
		SQLTableName:     opts.SQLTableName,
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

		wide := &ListOutputWide{
			Namespace:     opts.Namespace,
			AppID:         opts.AppID,
			Workflow:      resp.Name,
			InstanceID:    instanceID,
			Created:       resp.CreatedAt.AsTime(),
			LastUpdate:    resp.LastUpdatedAt.AsTime(),
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

	return listOutput, nil
}

// TODO: @joshvanl: use for `dapr workflow get-history xyz`
//func fetchHistory(ctx context.Context, cl client.Client, actorType, instanceID string) ([]*protos.HistoryEvent, error) {
//	var events []*protos.HistoryEvent
//	// Try starting from index 0, then 1 if no events found at 0
//	for startIndex := 0; startIndex <= 1; startIndex++ {
//		if len(events) > 0 {
//			break // Found events, no need to try next start index
//		}
//
//		for i := startIndex; i < maxHistoryEntries; i++ {
//			key := fmt.Sprintf("history-%06d", i)
//
//			resp, err := cl.GetActorState(ctx, &client.GetActorStateRequest{
//				ActorType: actorType,
//				ActorID:   instanceID,
//				KeyName:   key,
//			})
//			if err != nil {
//				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
//					return nil, err
//				}
//				break
//			}
//
//			if resp == nil || len(resp.Data) == 0 {
//				break
//			}
//
//			var event protos.HistoryEvent
//			if err = decodeKey(resp.Data, &event); err != nil {
//				return nil, fmt.Errorf("failed to decode history event %s: %w", key, err)
//			}
//
//			events = append(events, &event)
//		}
//	}
//
//	return events, nil
//}
//
//func decodeKey(data []byte, item proto.Message) error {
//	if len(data) == 0 {
//		return fmt.Errorf("empty value")
//	}
//
//	if err := protojson.Unmarshal(data, item); err == nil {
//		return nil
//	}
//
//	if unquoted, err := unquoteJSON(data); err == nil {
//		if err := protojson.Unmarshal([]byte(unquoted), item); err == nil {
//			return nil
//		}
//	}
//
//	if err := proto.Unmarshal(data, item); err == nil {
//		return nil
//	}
//
//	return fmt.Errorf("unable to decode history event (len=%d)", len(data))
//}
//
//func unquoteJSON(data []byte) (string, error) {
//	var s string
//	if err := json.Unmarshal(data, &s); err != nil {
//		return "", err
//	}
//	return s, nil
//}
//
//func runtimeStatus(status api.OrchestrationStatus) string {
//	switch status {
//	case api.RUNTIME_STATUS_RUNNING:
//		return "RUNNING"
//	case api.RUNTIME_STATUS_COMPLETED:
//		return "COMPLETED"
//	case api.RUNTIME_STATUS_CONTINUED_AS_NEW:
//		return "CONTINUED_AS_NEW"
//	case api.RUNTIME_STATUS_FAILED:
//		return "FAILED"
//	case api.RUNTIME_STATUS_CANCELED:
//		return "CANCELED"
//	case api.RUNTIME_STATUS_TERMINATED:
//		return "TERMINATED"
//	case api.RUNTIME_STATUS_PENDING:
//		return "PENDING"
//	case api.RUNTIME_STATUS_SUSPENDED:
//		return "SUSPENDED"
//	default:
//		return ""
//	}
//}
