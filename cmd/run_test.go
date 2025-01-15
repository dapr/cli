package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidateSchedulerHostAddress(t *testing.T) {
	t.Run("test scheduler host address - v1.14.0-rc.0", func(t *testing.T) {
		address := validateSchedulerHostAddress("1.14.0-rc.0", "")
		assert.Equal(t, "", address)
	})

	t.Run("test scheduler host address - v1.15.0-rc.0", func(t *testing.T) {
		address := validateSchedulerHostAddress("1.15.0", "")
		assert.Equal(t, "localhost:50006", address)
	})
}
