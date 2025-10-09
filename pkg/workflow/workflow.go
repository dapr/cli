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

	"github.com/dapr/cli/pkg/workflow/db"
	"github.com/dapr/cli/pkg/workflow/dclient"
)

type DBOptions struct {
	Namespace        string
	AppID            string
	Driver           string
	ConnectionString *string
	TableName        *string
}

func metakeys(ctx context.Context, opts DBOptions) ([]string, error) {
	if opts.ConnectionString == nil {
		return nil, fmt.Errorf("connection string is required for all drivers")
	}

	switch {
	case dclient.IsSQLDriver(opts.Driver):
		tableName := "state"
		if opts.TableName != nil {
			tableName = *opts.TableName
		}

		sqldb, err := db.SQL(ctx, opts.Driver, *opts.ConnectionString)
		if err != nil {
			return nil, err
		}
		defer sqldb.Close()

		return db.ListSQL(ctx, sqldb, tableName, db.ListOptions{
			Namespace: opts.Namespace,
			AppID:     opts.AppID,
		})

	case opts.Driver == "redis":
		client, err := db.Redis(ctx, *opts.ConnectionString)
		if err != nil {
			return nil, err
		}

		return db.ListRedis(ctx, client, db.ListOptions{
			Namespace: opts.Namespace,
			AppID:     opts.AppID,
		})

	case opts.Driver == "mongodb":
		client, err := db.Mongo(ctx, *opts.ConnectionString)
		if err != nil {
			return nil, err
		}

		collectionName := "daprCollection"
		if opts.TableName != nil {
			collectionName = *opts.TableName
		}

		return db.ListMongo(ctx, client.Database("daprStore"), collectionName, db.ListOptions{
			Namespace: opts.Namespace,
			AppID:     opts.AppID,
		})

	default:
		return nil, fmt.Errorf("unsupported driver: %s", opts.Driver)
	}
}
