package vmware

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/chaosnative/litmus-go/pkg/utils"
	"github.com/pkg/errors"
)

// UploadScript uploads a script file from the source to the target VM
func UploadScript(sourceFilePath, destinationFilePath, vmName, vmUserName, vmPassword string) error {

	command := fmt.Sprintf(`govc guest.upload -vm=%s -l=%s:%s "%s" "%s"`, vmName, vmUserName, vmPassword, sourceFilePath, destinationFilePath)
	_, stderr, err := utils.Shellout(command)

	if err != nil {
		return err
	}

	if stderr != "" {
		return errors.Errorf("%s", stderr)
	}

	return nil
}

// ExecuteScript executes a script file in the target VM
func ExecuteScript(destinationDir, fileName, timeoutSeconds, vmName, vmUserName, vmPassword string) (string, error) {

	command := fmt.Sprintf(`govc guest.run -vm=%s -l=%s:%s -C %s "echo %s | sudo -S timeout %s bash %s"`, vmName, vmUserName, vmPassword, destinationDir, vmPassword, timeoutSeconds, fileName)
	stdout, _, err := utils.Shellout(command)

	if err != nil {

		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 124 {
			return "", errors.Errorf("script execution timed out")

		} else {

			return "", err
		}
	}

	stdout = strings.TrimSuffix(stdout, "\n")

	return stdout, err
}

// UploadEnvs creates a file litmus_environment in the target VM which will contain all the environment variables
func UploadEnvs(destinationDir, envString, vmName, vmUserName, vmPassword string) error {

	command := fmt.Sprintf(`govc guest.run -vm=%s -l=%s:%s -C %s printf "%s" | tee -a litmus_environment`, vmName, vmUserName, vmPassword, destinationDir, envString)
	_, stderr, err := utils.Shellout(command)

	if err != nil {
		return err
	}

	if stderr != "" {
		return errors.Errorf("%s", stderr)
	}

	return nil
}
