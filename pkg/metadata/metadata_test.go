// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dapr/cli/pkg/api"
)

func TestMakeMetadataGetEndpoint(t *testing.T) {
	actual := makeMetadataGetEndpoint(9999)
	assert.Equal(t, fmt.Sprintf("http://127.0.0.1:9999/v%s/metadata", api.RuntimeAPIVersion), actual, "expected strings to match")
}
