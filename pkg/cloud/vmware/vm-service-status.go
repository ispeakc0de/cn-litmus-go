package vmware

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
)

func Shellout(command string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// GetService returns a boolean value indicating weather the given service exists on the given VM or not
func GetService(serviceName, vmName, datacenter, vmUserName, vmPassword string) (bool, error) {

	/*
		The following command displays the names of all the service units that systemd has attempted to parse and load into memory,
		in which we search for the given service name; stdout would always be of the format serviceName.service, if the service exists,
		and hence it has been omitted. If the service is not found, it will result into an error.
	*/
	command := fmt.Sprintf("govc guest.run -vm=%s -dc=%s -l=%s:%s systemctl list-unit-files --type service --no-page | awk '{print $1}' | grep %s.service", vmName, datacenter, vmUserName, vmPassword, serviceName)
	_, stderr, err := Shellout(command)

	if stderr != "" {
		return false, errors.Errorf("%s", stderr)
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// GetServiceState returns the state of a given service
func GetServiceState(serviceName, vmName, datacenter, vmUserName, vmPassWord string) (string, error) {

	/*
		The following command lists the property ActiveState of a given service in the format ActiveState=property,
		where 'property' can be any one of "active", "reloading", "inactive", "failed", "activating", and "deactivating".
	*/
	command := fmt.Sprintf("govc guest.run -vm=%s -dc=%s -l=%s:%s systemctl show %s -p ActiveState --no-page | sed 's/ActiveState=//g'", vmName, datacenter, vmUserName, vmPassWord, serviceName)
	stdout, stderr, err := Shellout(command)

	if stderr != "" {
		return "", errors.Errorf("%s", stderr)
	}

	if err != nil {
		return "", err
	}

	stdout = stdout[:len(stdout)-1]
	return stdout, nil
}
