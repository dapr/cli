package version

import (
	"os/exec"
	"runtime"
)

// GetRuntimeVersion returns the version for the local Actions runtime
func GetRuntimeVersion() string {
	runtimeName := ""
	if runtime.GOOS == "windows" {
		runtimeName = "actionsrt.exe"
	} else {
		runtimeName = "actionsrt"
	}

	out, err := exec.Command(runtimeName, "--version").Output()
	if err != nil {
		return "n/a"
	}
	return string(out)
}
