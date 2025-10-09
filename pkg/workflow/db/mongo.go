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
	"regexp"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func Mongo(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return client, nil
}

func ListMongo(ctx context.Context, db *mongo.Database, collection string, opts ListOptions) ([]string, error) {
	coll := db.Collection(collection)

	ns := regexp.QuoteMeta(opts.Namespace)
	app := regexp.QuoteMeta(opts.AppID)

	prefix := fmt.Sprintf("%s\\|\\|dapr\\.internal\\.%s\\.%s\\.workflow\\|\\|", app, ns, app)
	suffix := "\\|\\|metadata"
	regex := fmt.Sprintf("^%s.*%s$", prefix, suffix)

	filter := bson.M{
		"key": bson.M{
			"$regex":   regex,
			"$options": "",
		},
	}

	findOpts := options.Find().SetProjection(bson.M{"_id": 0, "key": 1})

	cur, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var keys []string
	for cur.Next(ctx) {
		var doc struct {
			Key string `bson:"key"`
		}
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		keys = append(keys, doc.Key)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}
