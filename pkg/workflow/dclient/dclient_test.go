/*
Copyright 2026 The Dapr Authors
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
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dapr/durabletask-go/api/protos"
	"github.com/dapr/durabletask-go/workflow"
)

type fakeSidecar struct {
	protos.UnimplementedTaskHubSidecarServiceServer
	listErr  error
	listResp *protos.ListInstanceIDsResponse
}

func (f *fakeSidecar) ListInstanceIDs(ctx context.Context, req *protos.ListInstanceIDsRequest) (*protos.ListInstanceIDsResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResp, nil
}

func newTestClient(t *testing.T, sidecar *fakeSidecar) *Client {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := grpc.NewServer()
	protos.RegisterTaskHubSidecarServiceServer(srv, sidecar)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)
	t.Cleanup(func() { lis.Close() })

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return &Client{
		WF:             workflow.NewClient(conn),
		kubernetesMode: true,
		appID:          "myapp",
		ns:             "default",
	}
}

func TestInstanceIDsSurfacesRuntimeError(t *testing.T) {
	// When the listing RPC fails and the state-store fallback cannot run, the
	// user must see the RPC error, not only the fallback's complaint.
	c := newTestClient(t, &fakeSidecar{
		listErr: errors.New("invalid key format: myapp||dapr.internal.default.myapp.workflow||weird||id||metadata"),
	})

	_, err := c.InstanceIDs(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key format")
	assert.Contains(t, err.Error(), "connection string is required")
}

func TestInstanceIDsSuccess(t *testing.T) {
	c := newTestClient(t, &fakeSidecar{
		listResp: &protos.ListInstanceIDsResponse{InstanceIds: []string{"wf-1", "wf-2"}},
	})

	ids, err := c.InstanceIDs(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{"wf-1", "wf-2"}, ids)
}

func TestInstanceIDsCancelledContext(t *testing.T) {
	// A cancelled context must surface immediately, without attempting the
	// state-store fallback.
	c := newTestClient(t, &fakeSidecar{
		listResp: &protos.ListInstanceIDsResponse{},
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := c.InstanceIDs(ctx)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "connection string is required")
}
