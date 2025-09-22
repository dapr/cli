package dclient

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/dapr/dapr/pkg/components/loader"
	"github.com/dapr/go-sdk/client"
	"github.com/dapr/kit/ptr"
)

type Client struct {
	Dapr             client.Client
	Cancel           context.CancelFunc
	StateStoreDriver string
	ConnectionString *string
	SQLTableName     *string
}

func DaprClient(ctx context.Context, kubernetesMode bool, namespace, appID string) (*Client, error) {
	client.SetLogger(nil)

	var client *Client
	var err error
	if kubernetesMode {
		client, err = kube(namespace, appID)
	} else {
		client, err = stand(ctx, appID)
	}

	return client, err
}

func stand(ctx context.Context, appID string) (*Client, error) {
	list, err := standalone.List()
	if err != nil {
		return nil, err
	}

	var proc *standalone.ListOutput
	for _, c := range list {
		if c.AppID == appID {
			proc = &c
			break
		}
	}

	if proc == nil {
		return nil, fmt.Errorf("Dapr app with id '%s' not found", appID)
	}

	comps, err := loader.NewLocalLoader(appID, proc.ResourcePaths).Load(ctx)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("no state store configured for app id %s", appID)
	}

	client, err := client.NewClientWithAddress("127.0.0.1:" + strconv.Itoa(proc.GRPCPort))
	if err != nil {
		return nil, err
	}

	driver, err := driverFromType(comp.Spec.Type)
	if err != nil {
		return nil, err
	}

	// Optimistically use the connection string in self-hosted mode.
	var connString *string
	var tableName *string
	for _, meta := range comp.Spec.Metadata {
		switch meta.Name {
		case "connectionString":
			connString = ptr.Of(meta.Value.String())
		case "tableName":
			tableName = ptr.Of(meta.Value.String())
		}
	}

	return &Client{
		Dapr:             client,
		Cancel:           func() {},
		StateStoreDriver: driver,
		ConnectionString: connString,
		SQLTableName:     tableName,
	}, nil
}

func kube(namespace string, appID string) (*Client, error) {
	list, err := kubernetes.List(namespace)
	if err != nil {
		return nil, err
	}

	var pod *kubernetes.ListOutput
	for _, p := range list {
		if p.AppID == appID {
			pod = &p
			break
		}
	}

	if pod == nil {
		return nil, fmt.Errorf("Dapr app with id '%s' not found in namespace %s", appID, namespace)
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
		namespace,
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

	kclient, err := kubernetes.DaprClient()
	if err != nil {
		return nil, err
	}
	comps, err := kubernetes.ListComponents(kclient, pod.Namespace)
	if err != nil {
		return nil, err
	}

	var comp *v1alpha1.Component
	for _, c := range comps.Items {
		for _, meta := range c.Spec.Metadata {
			if meta.Name == "actorStateStore" && meta.Value.String() == "true" {
				comp = &c
				break
			}
		}
	}

	if comp == nil {
		return nil, fmt.Errorf("no state store configured for app id %s", appID)
	}

	driver, err := driverFromType(comp.Spec.Type)
	if err != nil {
		return nil, err
	}

	client, err := client.NewClientWithAddress("localhost:" + pod.DaprGRPCPort)
	if err != nil {
		portForward.Stop()
		return nil, err
	}

	var tableName *string
	for _, meta := range comp.Spec.Metadata {
		switch meta.Name {
		case "tableName":
			tableName = ptr.Of(meta.Value.String())
		}
	}

	return &Client{
		Dapr:             client,
		Cancel:           portForward.Stop,
		StateStoreDriver: driver,
		SQLTableName:     tableName,
	}, nil
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
		return "", fmt.Errorf("unsupported state store type: %s")
	}
}

func IsSQLDriver(driver string) bool {
	return slices.Contains([]string{
		"mysql",
		"pgx",
		"sqlserver",
		"sqlite3",
		"oracle",
	}, driver)
}
