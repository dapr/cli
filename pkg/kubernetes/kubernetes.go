package kubernetes

import "os/exec"

const actionsManifestPath = "https://raw.githubusercontent.com/actions-org/actions/master/deploy/actions.yaml?token=AD4KK473CQJX24LS2MENQBS5CQTRQ"

func Init() error {
	err := runCmdAndWait("kubectl", "apply", "-f", actionsManifestPath)
	if err != nil {
		return err
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
