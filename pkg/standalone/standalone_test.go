// +build large_timeout
// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func tearDownInit(t *testing.T) {
	Uninstall(true, "")
}

func TestStandaloneInit(t *testing.T) {
	// Setup the teardown routine to cleanup
	defer tearDownInit(t)

	var err error

	t.Run("Init", func(t *testing.T) {
		err = Init("0.7.0", "", "")
	})

	assert.Nil(t, err)

	// Assert that default components folder is created at Init time
	defaultComponentsDir := getDefaultComponentsFolder()
	_, err = os.Stat(defaultComponentsDir)
	assert.Nil(t, err, "Default components directory not created at init time")
}
