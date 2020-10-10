// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package metadata

import (
	"fmt"
	"testing"

	"github.com/dapr/cli/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestMakeMetadataGetEndpoint(t *testing.T) {
	actual := makeMetadataGetEndpoint(9999)
	assert.Equal(t, fmt.Sprintf("http://127.0.0.1:9999/v%s/metadata", api.RuntimeAPIVersion), actual, "expected strings to match")
}
