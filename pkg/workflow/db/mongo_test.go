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

package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestListMongo(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("returns keys from _id field", func(mt *mtest.T) {
		key1 := "myapp||dapr.internal.default.myapp.workflow||instance-1||metadata"
		key2 := "myapp||dapr.internal.default.myapp.workflow||instance-2||metadata"

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "daprStore.daprCollection", mtest.FirstBatch,
			bson.D{{Key: "_id", Value: key1}},
		), mtest.CreateCursorResponse(0, "daprStore.daprCollection", mtest.NextBatch,
			bson.D{{Key: "_id", Value: key2}},
		))

		keys, err := ListMongo(mt.Context(), mt.DB, "daprCollection", ListOptions{
			Namespace: "default",
			AppID:     "myapp",
		})
		require.NoError(mt, err)
		assert.Equal(mt, []string{key1, key2}, keys)
	})

	mt.Run("returns nil when no documents match", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "daprStore.daprCollection", mtest.FirstBatch))

		keys, err := ListMongo(mt.Context(), mt.DB, "daprCollection", ListOptions{
			Namespace: "default",
			AppID:     "myapp",
		})
		require.NoError(mt, err)
		assert.Nil(mt, keys)
	})

	mt.Run("filter uses _id field with correct regex", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "daprStore.daprCollection", mtest.FirstBatch))

		_, err := ListMongo(mt.Context(), mt.DB, "daprCollection", ListOptions{
			Namespace: "default",
			AppID:     "myapp",
		})
		require.NoError(mt, err)

		// Verify the find command was sent with _id filter.
		cmd := mt.GetStartedEvent().Command
		filterVal := cmd.Lookup("filter")
		filterDoc := filterVal.Document()

		// The filter should contain an "_id" field, not "key".
		idVal, err := filterDoc.LookupErr("_id")
		assert.NoError(mt, err, "filter should query on _id field")
		_, err = filterDoc.LookupErr("key")
		assert.Error(mt, err, "filter should not query on key field")

		// Verify the $regex value matches the expected workflow metadata pattern.
		idDoc := idVal.Document()
		regexVal, err := idDoc.LookupErr("$regex")
		require.NoError(mt, err, "filter on _id should use $regex")
		assert.Equal(mt,
			`^myapp\|\|dapr\.internal\.default\.myapp\.workflow\|\|.*\|\|metadata$`,
			regexVal.StringValue(),
		)
	})
}
