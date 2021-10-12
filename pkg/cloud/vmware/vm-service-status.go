package vmware

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/utils/retry"
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

// GetService ensures that the given service exists on the given VM otherwise it returns an error
func GetService(serviceName, vmName, vmUserName, vmPassword string) error {

	/*
		The following command displays the names of all the service units that systemd has attempted to parse and load into memory,
		in which we search for the given service name; if the service exists, stdout would always be of the format serviceName.service,
		and hence it has been omitted, otherwise if the service is not found it will result into an error.
	*/
	command := fmt.Sprintf(`govc guest.run -vm=%s -l=%s:%s systemctl list-unit-files --type service --no-page | awk '{print $1}' | grep %s.service`, vmName, vmUserName, vmPassword, serviceName)
	_, stderr, err := Shellout(command)

	if stderr != "" {
		return errors.Errorf("%s", stderr)
	}

	if err != nil {
		return err
	}

	return nil
}

// GetServiceState returns the state of a given service
func GetServiceState(serviceName, vmName, vmUserName, vmPassword string) (string, error) {

	/*
		The following command lists the property ActiveState of a given service in the format ActiveState=property,
		where 'property' can be any one of "active", "reloading", "inactive", "failed", "activating", and "deactivating".
	*/
	command := fmt.Sprintf(`govc guest.run -vm=%s -l=%s:%s systemctl show %s -p ActiveState --no-page | sed 's/ActiveState=//g'`, vmName, vmUserName, vmPassword, serviceName)
	stdout, stderr, err := Shellout(command)

	if stderr != "" {
		return "", errors.Errorf("%s", stderr)
	}

	if err != nil {
		return "", err
	}

	// a newline character gets appendeed to the end of the string in stdout
	stdout = stdout[:len(stdout)-1]

	return stdout, nil
}

// ServiceStateCheck checks if the services exist on the VM and are in an active state
func ServiceStateCheck(serviceNames, vmName, vmUserName, vmPassword string) error {

	serviceNameList := strings.Split(serviceNames, ",")
	if len(serviceNameList) == 0 {
		return errors.Errorf("no service names provided")
	}

	if vmName == "" {
		return errors.Errorf("no vm name provided for the corresponding service names")
	}

	if vmUserName == "" || vmPassword == "" {
		return errors.Errorf("vm username or password is missing")
	}

	connectionState, powerState, err := GetVMState(vmName)
	if err != nil {
		return errors.Errorf("unable to fetch vm state, %s", err)
	}

	if connectionState != "connected" {
		return errors.Errorf("vm is not in connected state")
	}

	if powerState != "poweredOn" {
		return errors.Errorf("vm is not in powered-on state")
	}

	for _, serviceName := range serviceNameList {

		if err := GetService(serviceName, vmName, vmUserName, vmPassword); err != nil {
			return errors.Errorf("unable to find %s service, %s", serviceName, err)
		}

		serviceState, err := GetServiceState(serviceName, vmName, vmUserName, vmPassword)
		if err != nil {
			return errors.Errorf("failed to get %s service state", serviceName)
		}

		if serviceState != "active" {
			return errors.Errorf("%s service is not in active state", serviceName)
		}
	}

	return nil
}

// WaitForServiceStop will wait for the service to completely stop
func WaitForServiceStop(vcenterServer, vmName, serviceName, vmUserName, vmPassword string, delay, timeout int) error {

	log.Infof("[Status]: Checking service %s status for stopping", serviceName)
	return retry.
		Times(uint(timeout / delay)).
		Wait(time.Duration(delay) * time.Second).
		Try(func(attempt uint) error {

			serviceState, err := GetServiceState(serviceName, vmName, vmUserName, vmPassword)
			if err != nil {
				return errors.Errorf("failed to get the service state")
			}

			if serviceState != "inactive" {
				log.Infof("[Info]: The service state is %v", serviceState)
				return errors.Errorf("service is not yet in inactive state")
			}

			log.Infof("[Info]: The service state is %v", serviceState)
			return nil
		})
}

// WaitForServiceStart will wait for the service to completely start
func WaitForServiceStart(vcenterServer, vmName, serviceName, vmUserName, vmPassword string, delay, timeout int) error {

	log.Infof("[Status]: Checking service %s status for stopping", serviceName)
	return retry.
		Times(uint(timeout / delay)).
		Wait(time.Duration(delay) * time.Second).
		Try(func(attempt uint) error {

			serviceState, err := GetServiceState(serviceName, vmName, vmUserName, vmPassword)
			if err != nil {
				return errors.Errorf("failed to get the service state")
			}

			if serviceState != "active" {
				log.Infof("[Info]: The service state is %v", serviceState)
				return errors.Errorf("service is not yet in inactive state")
			}

			log.Infof("[Info]: The service state is %v", serviceState)
			return nil
		})
}
