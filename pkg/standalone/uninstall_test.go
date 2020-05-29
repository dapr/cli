// +build large_timeout
// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package standalone

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupUninstall(t *testing.T) {
	err := Init("0.7.0", "", "")
	assert.Nil(t, err, "Unable to setup the test properly")
}

func tearDownUninstall(t *testing.T) {
	fmt.Println("teardown ***")
	Uninstall(true, "")
}

func TestUninstall(t *testing.T) {
	setupUninstall(t)
	// Setup the teardown routine to cleanup
	defer tearDownUninstall(t)

	var err error

	t.Run("Uninstall", func(t *testing.T) {
		err = Uninstall(true, "")
	})

	assert.Nil(t, err)

	// Assert that default components folder is deleted after uninstall
	defaultComponentsDir := getDefaultComponentsFolder()
	_, err = os.Stat(defaultComponentsDir)
	assert.NotNil(t, err, "Default components directory not deleted after uninstall")
}
