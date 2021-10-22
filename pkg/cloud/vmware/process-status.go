package vmware

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chaosnative/litmus-go/pkg/utils"
	"github.com/pkg/errors"
)

// GetProcess checks if the given process exists on a VM or not
func GetProcess(processId, vmName, vmUserName, vmPassword string) (bool, error) {

	type Process struct {
		ProcessInfo []struct {
			Name string `json:"Name"`
			Pid  string `json:"Pid"`
		} `json:"ProcessInfo"`
	}

	command := fmt.Sprintf(`govc guest.ps -vm %s -l %s:%s -json=true -p %s`, vmName, vmUserName, vmPassword, processId)

	stdout, stderr, err := utils.Shellout(command)
	if err != nil {
		return false, err
	}

	if stderr != "" {
		return false, errors.Errorf("%s", stderr)
	}

	var processDetails Process
	json.Unmarshal([]byte(stdout), &processDetails)

	if len(processDetails.ProcessInfo) != 1 {
		return false, nil
	}

	return true, nil
}

// ProcessStateCheck validates that all the given processes are alive
func ProcessStateCheck(vmName, processIds, vmUserName, vmPassword string) error {

	if vmName == "" {
		return errors.Errorf("no vm name provided")
	}

	if err := GetVM(vmName); err != nil {
		return errors.Errorf("failed to get vm %s, err: %s", vmName, err.Error())
	}

	if vmUserName == "" {
		return errors.Errorf("no vm username provided")
	}

	if vmPassword == "" {
		return errors.Errorf("no vm password provided")
	}

	connectionState, powerState, err := GetVMState(vmName)
	if err != nil {
		return errors.Errorf("unable to fetch vm %s state, %s", vmName, err.Error())
	}

	if connectionState != "connected" {
		return errors.Errorf("vm %s is not in connected state", vmName)
	}

	if powerState != "poweredOn" {
		return errors.Errorf("vm %s is not in powered-on state", vmName)
	}

	processIdList := strings.Split(processIds, ",")
	if len(processIdList) == 0 {
		return errors.Errorf("no process id provided")
	}

	for _, processId := range processIdList {

		isProcessAlive, err := GetProcess(processId, vmName, vmUserName, vmPassword)
		if err != nil {
			return errors.Errorf("failed to get process %s, err: %s", processId, err.Error())
		}

		if !isProcessAlive {
			return errors.Errorf("process %s not found", processId)
		}
	}

	return nil
}
