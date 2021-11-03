package vmware

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chaosnative/litmus-go/pkg/utils"
	"github.com/pkg/errors"
)

// GetVM checks if a given vm exists or not
func GetVM(vmName string) error {

	command := fmt.Sprintf(`govc find vm -name %s`, vmName)
	stdout, stderr, err := utils.Shellout(command)

	if stderr != "" {
		return errors.Errorf("%s", stderr)
	}

	if err != nil {
		return err
	}

	if stdout == "" {
		return errors.Errorf("vm not found")
	}

	return nil
}

// GetVMStatus returns the connection state and power state of a given VM
func GetVMState(vmName string) (string, string, error) {

	type VMStatus struct {
		VirtualMachines []struct {
			Summary struct {
				Runtime struct {
					ConnectionState string `json:"ConnectionState"`
					PowerState      string `json:"PowerState"`
				} `json:"Runtime"`
			} `json:"Summary"`
		} `json:"VirtualMachines"`
	}

	command := fmt.Sprintf(`govc vm.info -json %s`, vmName)
	stdout, stderr, err := utils.Shellout(command)

	if stderr != "" {
		return "", "", errors.Errorf("%s", stderr)
	}

	if err != nil {
		return "", "", err
	}

	var vmStatus VMStatus
	json.Unmarshal([]byte(stdout), &vmStatus)

	return vmStatus.VirtualMachines[0].Summary.Runtime.ConnectionState, vmStatus.VirtualMachines[0].Summary.Runtime.PowerState, nil
}

func VMStateCheck(vmNames string) error {

	vmNameList := strings.Split(vmNames, ",")
	if len(vmNameList) == 0 {
		return errors.Errorf("no vm name found")
	}

	for _, vmName := range vmNameList {

		if err := GetVM(vmName); err != nil {
			return errors.Errorf("vm %s not found, err: %s", vmName, err.Error())
		}

		connectionState, powerState, err := GetVMState(vmName)
		if err != nil {
			return errors.Errorf("unable to fetch vm %s state, err: %s", vmName, err.Error())
		}

		if connectionState != "connected" {
			return errors.Errorf("vm %s is not in connected state", vmName)
		}

		if powerState != "poweredOn" {
			return errors.Errorf("vm %s is not in powered-on state", vmName)
		}
	}

	return nil
}
