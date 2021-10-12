package vmware

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

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
	stdout, stderr, err := Shellout(command)

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