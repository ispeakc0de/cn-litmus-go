package vmware

import (
	"encoding/json"
	"fmt"

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
