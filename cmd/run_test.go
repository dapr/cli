package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestStartLocalServicesFlag(t *testing.T) {
	flag := RunCmd.Flags().Lookup("start-local-services")
	require.NotNil(t, flag)

	t.Run("default is true", func(t *testing.T) {
		assert.Equal(t, "bool", flag.Value.Type())
		assert.Equal(t, "true", flag.DefValue)
		assert.True(t, startLocalServices)
	})

	t.Run("set to false", func(t *testing.T) {
		originalRunFilePath := runFilePath
		t.Cleanup(func() {
			runFilePath = originalRunFilePath
			require.NoError(t, flag.Value.Set(flag.DefValue))
			flag.Changed = false
		})

		require.NoError(t, RunCmd.Flags().Parse([]string{"--start-local-services=false"}))
		assert.False(t, startLocalServices)
		assert.True(t, flag.Changed)

		// With --run-file, the run file sets this value, as for the address flags.
		runFilePath = "some/path"
		assert.Contains(t, detectIncompatibleFlags(RunCmd), "start-local-services")
	})
}

func TestLocalServicesToStart(t *testing.T) {
	testcases := []struct {
		name          string
		enabled       bool
		placementSet  bool
		schedulerSet  bool
		wantPlacement bool
		wantScheduler bool
	}{
		{name: "enabled, no address", enabled: true, wantPlacement: true, wantScheduler: true},
		{name: "enabled, placement address", enabled: true, placementSet: true, wantScheduler: true},
		{name: "enabled, scheduler address", enabled: true, schedulerSet: true, wantPlacement: true},
		{name: "enabled, both addresses", enabled: true, placementSet: true, schedulerSet: true},
		{name: "disabled, no address", enabled: false},
		{name: "disabled, both addresses", enabled: false, placementSet: true, schedulerSet: true},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			placement, scheduler := localServicesToStart(tc.enabled, tc.placementSet, tc.schedulerSet)
			assert.Equal(t, tc.wantPlacement, placement, "placement")
			assert.Equal(t, tc.wantScheduler, scheduler, "scheduler")
		})
	}
}
