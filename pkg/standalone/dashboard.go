package standalone

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/dapr/cli/pkg/print"
)

// RunDashboard finds the dashboard binary and runs it
func RunDashboard() {
	// Use the default binary install location
	dashboardPath := defaultDaprBinPath()
	binaryName := "dashboard"
	if runtime.GOOS == "windows" {
		binaryName = "dashboard.exe"
	}

	// Construct command to run dashboard
	cmdDashboardStandalone := &exec.Cmd{
		Path:   filepath.Join(dashboardPath, binaryName),
		Dir:    dashboardPath,
		Stdout: os.Stdout,
	}

	err := cmdDashboardStandalone.Run()
	if err != nil {
		print.FailureStatusEvent(os.Stdout, "Dapr dashboard not found. Is Dapr installed?")
	}
}
