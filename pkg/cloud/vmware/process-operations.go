package vmware

import (
	"fmt"

	"github.com/chaosnative/litmus-go/pkg/utils"
)

// KillLinuxProcess kills a given process in a Linux VM
func KillLinuxProcess(pid, vmName, vmUserName, vmPassword string) error {

	command := fmt.Sprintf(`govc guest.run -vm=%s -l=%s:%s "echo %s | sudo -S kill %s"`, vmName, vmUserName, vmPassword, vmPassword, pid)

	_, _, err := utils.Shellout(command)
	if err != nil {
		return err
	}

	return nil
}
