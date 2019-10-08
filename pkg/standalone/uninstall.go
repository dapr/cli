package standalone

import (
	"errors"

	"github.com/dapr/cli/utils"
)

func Uninstall() error {
	err := utils.RunCmdAndWait(
		"docker", "rm",
		"--force",
		DaprPlacementContainerName)
	if err != nil {
		return errors.New("Dapr Placement Container may not exist")
	}
	return nil
}
