package kubernetes

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/actionscore/cli/pkg/print"

	"github.com/briandowns/spinner"
)

const actionsManifestPath = "https://actionsreleases.blob.core.windows.net/manifest/actions-operator.yaml"

func Init() error {
	msg := "Deploying the Actions Operator to your cluster..."
	var s *spinner.Spinner

	if runtime.GOOS == "windows" {
		print.InfoStatusEvent(os.Stdout, msg)
	} else {
		s = spinner.New(spinner.CharSets[1], 100*time.Millisecond)
		s.Writer = os.Stdout
		s.Color("blue")
		s.Suffix = fmt.Sprintf("  %s", msg)
		s.Start()
	}

	err := runCmdAndWait("kubectl", "apply", "-f", actionsManifestPath)
	if err != nil {
		if s != nil {
			s.Stop()
		}
		return err
	}

	if s != nil {
		s.Stop()
		print.SuccessStatusEvent(os.Stdout, msg)
	}
	return nil
}

func runCmdAndWait(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	err := cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
