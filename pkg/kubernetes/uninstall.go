package kubernetes

import "errors"

func Uninstall() error {
	err := runCmdAndWait("kubectl", "delete", "-f", actionsManifestPath)
	if err != nil {
		return errors.New("Is Actions running? Please note uninstall does not remove Actions when installed via Helm")
	}
	return nil
}
