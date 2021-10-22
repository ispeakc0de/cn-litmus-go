package vmware

import (
	"fmt"
	"time"

	"github.com/chaosnative/litmus-go/pkg/utils"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/utils/retry"
	"github.com/pkg/errors"
)

// KillLinuxProcess kills a given process in a Linux VM
func KillLinuxProcess(processId, vmName, vmUserName, vmPassword string) error {

	command := fmt.Sprintf(`govc guest.run -vm=%s -l=%s:%s "echo %s | sudo -S kill %s"`, vmName, vmUserName, vmPassword, vmPassword, processId)

	_, _, err := utils.Shellout(command)
	if err != nil {
		return err
	}

	return nil
}

// WaitForProcessKill will wait for the given process to completely kill
func WaitForProcessKill(processId, vmName, vmUserName, vmPassword string, delay, timeout int) error {

	log.Infof("[Status]: Checking for process %s", processId)
	return retry.
		Times(uint(timeout / delay)).
		Wait(time.Duration(delay) * time.Second).
		Try(func(attempt uint) error {

			isProcessAlive, err := GetProcess(processId, vmName, vmUserName, vmPassword)
			if err != nil {
				return errors.Errorf("failed to get process %s, err: %s", processId, err.Error())
			}

			if isProcessAlive {
				log.Infof("[Info]: Process %s is alive", processId)
				return errors.Errorf("process %s is not yet killed", processId)
			}

			log.Infof("[Info]: Process %s is killed", processId)
			return nil
		})
}
