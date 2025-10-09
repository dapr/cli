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
	"fmt"

	"github.com/redis/go-redis/v9"
)

func Redis(ctx context.Context, url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err

	}

	rdb := redis.NewClient(opt)
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}

func ListRedis(ctx context.Context, rdb *redis.Client, opts ListOptions) ([]string, error) {
	pattern := fmt.Sprintf("%s||dapr.internal.%s.%s.workflow||*||metadata",
		opts.AppID, opts.Namespace, opts.AppID)

	var (
		cursor uint64
		keys   []string
	)

	const scanCount int64 = 1000

	for {
		res, nextCursor, err := rdb.Scan(ctx, cursor, pattern, scanCount).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, res...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return keys, nil
}
