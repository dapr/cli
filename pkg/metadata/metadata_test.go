/*
Copyright 2021 The Dapr Authors
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

package metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dapr/cli/pkg/api"
)

func TestMakeMetadataGetEndpoint(t *testing.T) {
	t.Parallel()
	actual := makeMetadataGetEndpoint(9999)
	assert.Equal(t, fmt.Sprintf("http://127.0.0.1:9999/v%s/metadata", api.RuntimeAPIVersion), actual, "expected strings to match")
}
