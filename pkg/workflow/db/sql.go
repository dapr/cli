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

package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
)

func SQL(ctx context.Context, driver, connString string) (*sql.DB, error) {
	db, err := sql.Open(driver, connString)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

func ListSQL(ctx context.Context, db *sql.DB, table string, opts ListOptions) ([]string, error) {
	query := fmt.Sprintf(`SELECT key FROM "%s" WHERE key LIKE ?;`, table)
	like := opts.AppID + "||dapr.internal." + opts.Namespace + "." + opts.AppID + ".workflow||%||metadata"

	rows, err := db.QueryContext(ctx, query, like)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}

		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}
