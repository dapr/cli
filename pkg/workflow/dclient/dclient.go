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

package dclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/dapr/cli/pkg/workflow/db"
	"github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/dapr/dapr/pkg/components/loader"
	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/durabletask-go/workflow"
	"github.com/dapr/go-sdk/client"
	"github.com/dapr/kit/ptr"
)

const maxHistoryEntries = 1000

type Options struct {
	KubernetesMode     bool
	Namespace          string
	AppID              string
	RuntimePath        string
	DBConnectionString *string
}

type Client struct {
	Dapr   client.Client
	WF     *workflow.Client
	Cancel context.CancelFunc

	kubernetesMode bool
	resourcePaths  []string
	appID          string
	ns             string
	dbConnString   *string
}

func DaprClient(ctx context.Context, opts Options) (*Client, error) {
	client.SetLogger(nil)

	var client *Client
	var err error
	if opts.KubernetesMode {
		client, err = kube(ctx, opts)
	} else {
		client, err = stand(ctx, opts)
	}

	return client, err
}

func stand(ctx context.Context, opts Options) (*Client, error) {
	list, err := standalone.List()
	if err != nil {
		return nil, err
	}

	var proc *standalone.ListOutput
	for _, c := range list {
		if c.AppID == opts.AppID {
			proc = &c
			break
		}
	}

	if proc == nil {
		return nil, fmt.Errorf("Dapr app with id '%s' not found", opts.AppID)
	}

	resourcePaths := proc.ResourcePaths
	if len(resourcePaths) == 0 {
		var daprDirPath string
		daprDirPath, err = standalone.GetDaprRuntimePath(opts.RuntimePath)
		if err != nil {
			return nil, err
		}

		resourcePaths = []string{standalone.GetDaprComponentsPath(daprDirPath)}
	}

	client, err := client.NewClientWithAddress("localhost:" + strconv.Itoa(proc.GRPCPort))
	if err != nil {
		return nil, err
	}

	//nolint:staticcheck
	conn, err := grpc.DialContext(ctx, "localhost:"+strconv.Itoa(proc.GRPCPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		Dapr:           client,
		WF:             workflow.NewClient(conn),
		Cancel:         func() { conn.Close() },
		kubernetesMode: false,
		resourcePaths:  resourcePaths,
		appID:          opts.AppID,
		ns:             opts.Namespace,
		dbConnString:   opts.DBConnectionString,
	}, nil
}

func kube(ctx context.Context, opts Options) (*Client, error) {
	list, err := kubernetes.List(opts.Namespace)
	if err != nil {
		return nil, err
	}

	var pod *kubernetes.ListOutput
	for _, p := range list {
		if p.AppID == opts.AppID {
			pod = &p
			break
		}
	}

	if pod == nil {
		return nil, fmt.Errorf("Dapr app with id '%s' not found in namespace %s", opts.AppID, opts.Namespace)
	}

	config, _, err := kubernetes.GetKubeConfigClient()
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(pod.DaprGRPCPort)
	if err != nil {
		return nil, err
	}

	portForward, err := kubernetes.NewPortForward(
		config,
		opts.Namespace,
		pod.PodName,
		"localhost",
		port,
		port,
		false,
	)
	if err != nil {
		return nil, err
	}

	if err = portForward.Init(); err != nil {
		return nil, err
	}

	client, err := client.NewClientWithAddress("localhost:" + strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	//nolint:staticcheck
	conn, err := grpc.DialContext(ctx, "localhost:"+strconv.Itoa(port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		WF:             workflow.NewClient(conn),
		Dapr:           client,
		Cancel:         func() { conn.Close(); portForward.Stop() },
		kubernetesMode: true,
		appID:          opts.AppID,
		ns:             opts.Namespace,
		dbConnString:   opts.DBConnectionString,
	}, nil
}

func (c *Client) InstanceIDs(ctx context.Context) ([]string, error) {
	resp, err := c.WF.ListInstanceIDs(ctx)
	if err != nil {
		code, ok := status.FromError(err)
		if !ok || (code.Code() != codes.Unimplemented && code.Code() != codes.Unknown) {
			return nil, err
		}

		// Dapr is pre v1.17, so fall back to reading from the state store
		// directly.
		var metaKeys []string
		metaKeys, err = c.metaKeysFromDB(ctx)
		if err != nil {
			return nil, err
		}

		instanceIDs := make([]string, 0, len(metaKeys))
		for _, key := range metaKeys {
			split := strings.Split(key, "||")
			if len(split) != 4 {
				continue
			}

			instanceIDs = append(instanceIDs, split[2])
		}

		return instanceIDs, err
	}

	ids := resp.InstanceIds

	for resp.ContinuationToken != nil {
		resp, err = c.WF.ListInstanceIDs(ctx, workflow.WithListInstanceIDsContinuationToken(*resp.ContinuationToken))
		if err != nil {
			return nil, err
		}

		ids = append(ids, resp.InstanceIds...)
	}

	return ids, nil
}

func (c *Client) InstanceHistory(ctx context.Context, instanceID string) ([]*protos.HistoryEvent, error) {
	var history []*protos.HistoryEvent
	resp, err := c.WF.GetInstanceHistory(ctx, instanceID)
	if err != nil {
		code, ok := status.FromError(err)
		if !ok || (code.Code() != codes.Unimplemented && code.Code() != codes.Unknown) {
			return nil, err
		}

		// Dapr is pre v1.17, so fall back to reading from the state store
		// directly.
		history, err = c.fetchHistory(ctx, instanceID)
		if err != nil {
			return nil, err
		}
	} else {
		history = resp.Events
	}

	// Sort: EventId if both present, else Timestamp
	sort.SliceStable(history, func(i, j int) bool {
		ei, ej := history[i], history[j]
		if ei.EventId > 0 && ej.EventId > 0 {
			return ei.EventId < ej.EventId
		}
		ti, tj := ei.GetTimestamp().AsTime(), ej.GetTimestamp().AsTime()
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return ei.EventId < ej.EventId
	})

	return history, nil
}

func (c *Client) metaKeysFromDB(ctx context.Context) ([]string, error) {
	if c.dbConnString == nil {
		return nil, fmt.Errorf("connection string is required for all database drivers for Dapr pre v1.17")
	}

	var comps []v1alpha1.Component
	if c.kubernetesMode {
		kclient, err := kubernetes.DaprClient()
		if err != nil {
			return nil, err
		}

		kcomps, err := kubernetes.ListComponents(kclient, c.ns)
		if err != nil {
			return nil, err
		}
		comps = kcomps.Items
	} else {
		var err error
		comps, err = loader.NewLocalLoader(c.appID, c.resourcePaths).Load(ctx)
		if err != nil {
			return nil, err
		}
	}

	var comp *v1alpha1.Component
	for _, c := range comps {
		for _, meta := range c.Spec.Metadata {
			if meta.Name == "actorStateStore" && meta.Value.String() == "true" {
				comp = &c
				break
			}
		}
	}

	if comp == nil {
		return nil, fmt.Errorf("no actor state store configured for app id %s", c.appID)
	}

	driver, err := driverFromType(comp.Spec.Type)
	if err != nil {
		return nil, err
	}

	var tableName *string
	for _, meta := range comp.Spec.Metadata {
		switch meta.Name {
		case "tableName":
			tableName = ptr.Of(meta.Value.String())
		}
	}

	switch {
	case isSQLDriver(driver):
		if tableName == nil {
			tableName = ptr.Of("state")
		}

		sqldb, err := db.SQL(ctx, driver, *c.dbConnString)
		if err != nil {
			return nil, err
		}
		defer sqldb.Close()

		key := "key"
		if driver == "mysql" {
			key = "id"
		}

		return db.ListSQL(ctx, sqldb, *tableName, key, db.ListOptions{
			Namespace: c.ns,
			AppID:     c.appID,
		})

	case driver == "redis":
		client, err := db.Redis(ctx, *c.dbConnString)
		if err != nil {
			return nil, err
		}

		return db.ListRedis(ctx, client, db.ListOptions{
			Namespace: c.ns,
			AppID:     c.appID,
		})

	case driver == "mongodb":
		client, err := db.Mongo(ctx, *c.dbConnString)
		if err != nil {
			return nil, err
		}

		collectionName := "daprCollection"
		if tableName != nil {
			collectionName = *tableName
		}

		return db.ListMongo(ctx, client.Database("daprStore"), collectionName, db.ListOptions{
			Namespace: c.ns,
			AppID:     c.appID,
		})

	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
}

func driverFromType(v string) (string, error) {
	switch v {
	case "state.mysql":
		return "mysql", nil
	case "state.postgresql":
		return "pgx", nil
	case "state.sqlserver":
		return "sqlserver", nil
	case "state.sqlite":
		return "sqlite3", nil
	case "state.oracledatabase":
		return "oracle", nil
	case "state.cockroachdb":
		return "pgx", nil
	case "state.redis":
		return "redis", nil
	case "state.mongodb":
		return "mongodb", nil
	default:
		return "", fmt.Errorf("unsupported state store type: %s", v)
	}
}

func isSQLDriver(driver string) bool {
	return slices.Contains([]string{
		"mysql",
		"pgx",
		"sqlserver",
		"sqlite3",
		"oracle",
	}, driver)
}

func (c *Client) fetchHistory(ctx context.Context, instanceID string) ([]*protos.HistoryEvent, error) {

	actorType := "dapr.internal." + c.ns + "." + c.appID + ".workflow"

	var events []*protos.HistoryEvent
	for startIndex := 0; startIndex <= 1; startIndex++ {
		if len(events) > 0 {
			break
		}

		for i := startIndex; i < maxHistoryEntries; i++ {
			key := fmt.Sprintf("history-%06d", i)

			resp, err := c.Dapr.GetActorState(ctx, &client.GetActorStateRequest{
				ActorType: actorType,
				ActorID:   instanceID,
				KeyName:   key,
			})
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					return nil, err
				}
				break
			}

			if resp == nil || len(resp.Data) == 0 {
				break
			}

			var event protos.HistoryEvent
			if err = decodeKey(resp.Data, &event); err != nil {
				return nil, fmt.Errorf("failed to decode history event %s: %w", key, err)
			}

			events = append(events, &event)
		}
	}

	return events, nil
}

func decodeKey(data []byte, item proto.Message) error {
	if len(data) == 0 {
		return fmt.Errorf("empty value")
	}

	if err := protojson.Unmarshal(data, item); err == nil {
		return nil
	}

	if unquoted, err := UnquoteJSON(data); err == nil {
		if err := protojson.Unmarshal([]byte(unquoted), item); err == nil {
			return nil
		}
	}

	if err := proto.Unmarshal(data, item); err == nil {
		return nil
	}

	return fmt.Errorf("unable to decode history event (len=%d)", len(data))
}

func UnquoteJSON(data []byte) (string, error) {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return "", err
	}
	return s, nil
}
