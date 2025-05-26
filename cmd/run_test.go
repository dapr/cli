package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
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

func TestDetectIncompatibleFlags(t *testing.T) {
	// Setup a temporary run file path to trigger the incompatible flag check
	originalRunFilePath := runFilePath
	runFilePath = "some/path"
	defer func() {
		// Restore the original runFilePath
		runFilePath = originalRunFilePath
	}()

	t.Run("detect incompatible flags", func(t *testing.T) {
		// Create a test command with flags
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("app-id", "", "")
		cmd.Flags().String("dapr-http-port", "", "")
		cmd.Flags().String("kubernetes", "", "")   // Compatible flag
		cmd.Flags().String("runtime-path", "", "") // Compatible flag
		cmd.Flags().String("log-as-json", "", "")  // Compatible flag

		// Mark flags as changed
		cmd.Flags().Set("app-id", "myapp")
		cmd.Flags().Set("dapr-http-port", "3500")
		cmd.Flags().Set("kubernetes", "true")
		cmd.Flags().Set("runtime-path", "/path/to/runtime")
		cmd.Flags().Set("log-as-json", "true")

		// Test detection
		incompatibleFlags := detectIncompatibleFlags(cmd)
		assert.Len(t, incompatibleFlags, 2)
		assert.Contains(t, incompatibleFlags, "app-id")
		assert.Contains(t, incompatibleFlags, "dapr-http-port")
		assert.NotContains(t, incompatibleFlags, "kubernetes")
		assert.NotContains(t, incompatibleFlags, "runtime-path")
		assert.NotContains(t, incompatibleFlags, "log-as-json")
	})

	t.Run("no incompatible flags when run file not specified", func(t *testing.T) {
		// Create a test command with flags
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("app-id", "", "")
		cmd.Flags().String("dapr-http-port", "", "")

		// Mark flags as changed
		cmd.Flags().Set("app-id", "myapp")
		cmd.Flags().Set("dapr-http-port", "3500")

		// Temporarily clear runFilePath
		originalRunFilePath := runFilePath
		runFilePath = ""
		defer func() {
			runFilePath = originalRunFilePath
		}()

		// Test detection
		incompatibleFlags := detectIncompatibleFlags(cmd)
		assert.Nil(t, incompatibleFlags)
	})
}
