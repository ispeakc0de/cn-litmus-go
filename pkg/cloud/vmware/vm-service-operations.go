package vmware

import "fmt"

// StopService stops a given service in a given VM
func StopService(serviceName, vmName, vmUserName, vmPassword string) error {

	command := fmt.Sprintf(`govc guest.run -vm=%s -l=%s:%s printf "%s" | sudo -S systemctl stop %s`, vmName, vmUserName, vmPassword, vmPassword, serviceName)
	_, _, err := Shellout(command)

	if err != nil {
		return err
	}

	return nil
}

// StartService starts a given service in a given VM
func StartService(serviceName, vmName, vmUserName, vmPassword string) error {

	command := fmt.Sprintf(`govc guest.run -vm=%s -l=%s:%s printf "%s" | sudo -S systemctl start %s`, vmName, vmUserName, vmPassword, vmPassword, serviceName)
	_, _, err := Shellout(command)

	if err != nil {
		return err
	}

	return nil
}
