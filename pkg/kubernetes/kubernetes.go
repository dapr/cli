package kubernetes

import (
	"os"
	"os/exec"
	"time"

	"github.com/actionscore/cli/pkg/print"

	"github.com/briandowns/spinner"
)

const actionsManifestPath = "https://actionsreleases.blob.core.windows.net/manifest/actions-operator.yaml"

func Init() error {
	s := spinner.New(spinner.CharSets[1], 100*time.Millisecond)
	s.Writer = os.Stdout
	s.Color("blue")
	s.Suffix = "  Deploying the Actions Operator to your cluster... "
	s.Start()

	err := runCmdAndWait("kubectl", "apply", "-f", actionsManifestPath)
	if err != nil {
		s.Stop()
		return err
	}

	s.Stop()
	print.SuccessStatusEvent(os.Stdout, "Deploying the Actions Operator to your cluster...")
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
